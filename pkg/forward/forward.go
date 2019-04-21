package forward

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/webhookrelay/relay-go/pkg/retryablehttp"
	"github.com/webhookrelay/relay-go/pkg/types"
	"go.uber.org/zap"
)

// Relayer - relayer interface
type Relayer interface {
	Relay(wh *types.Event) *types.EventStatus
}

// DefaultRelayer - default 'last mile' webhook relayer
type DefaultRelayer struct {
	hClient *http.Client

	rClient *retryablehttp.Client

	// maximum number of retries that this relayer should try
	// before giving up.
	retries int

	// backoff strategy in seconds
	backoff int

	logger *zap.SugaredLogger
}

// Opts - configuration
type Opts struct {
	Retries  int
	Insecure bool
	Logger   *zap.SugaredLogger
}

// NewDefaultRelayer - create an instance of default relayer
func NewDefaultRelayer(opts *Opts) *DefaultRelayer {

	httpClient := &http.Client{}

	if opts.Logger == nil {
		cfg := zap.NewProductionConfig()
		cfg.DisableCaller = true
		cfg.DisableStacktrace = true
		cfg.Encoding = "console"

		l, err := cfg.Build()
		if err != nil {
			panic("failed to initialise logger")
		}
		opts.Logger = l.Sugar()
	}

	client := retryablehttp.NewClient(opts.Logger)

	if opts.Insecure {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		insecureClient := &http.Client{Transport: tr}
		client.HTTPClient = insecureClient
		httpClient = insecureClient
	}

	client.RetryMax = opts.Retries

	return &DefaultRelayer{rClient: client, hClient: httpClient, logger: opts.Logger}
}

// Relay - relaying incoming webhook to original destination
func (r *DefaultRelayer) Relay(wh types.Event) *types.EventStatus {
	if wh.RawQuery != "" {
		wh.Meta.OutputDestination = wh.Meta.OutputDestination + "?" + wh.RawQuery
	}
	req, err := retryablehttp.NewRequest(wh.Method, wh.Meta.OutputDestination, bytes.NewReader([]byte(wh.Body)))
	if err != nil {
		return &types.EventStatus{
			ID:         wh.Meta.ID,
			StatusCode: 0,
			Message:    fmt.Sprintf("invalid request: %s", err),
		}
	}

	var retries int
	var statusCode int

	req.Header = wh.Headers

	resp, err := r.rClient.Do(req)
	if resp != nil {
		retries = retryablehttp.GetRetries(resp)
		statusCode = resp.StatusCode

	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return &types.EventStatus{
			ID:         wh.Meta.ID,
			StatusCode: statusCode,
			Message:    fmt.Sprintf("invalid request: %s", err),
			Retries:    retries,
		}
	}

	var bodyStr string
	if resp.StatusCode > 399 {
		body, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			bodyStr = string(body)
		}

		r.logger.Warnw("unexpected status code",
			"status_code", resp.StatusCode,
			"destination", wh.Meta.OutputDestination,
			"method", wh.Method,
			"body", bodyStr,
		)
	}

	return &types.EventStatus{
		ID:         wh.Meta.ID,
		StatusCode: resp.StatusCode,
		Message:    bodyStr,
		Retries:    retries,
	}
}
