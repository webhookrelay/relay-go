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

// Forwarder is responsible for receiving and processing incoming webhook events
type Forwarder interface {
	Forward(wh types.Event) *types.EventStatus
}

var _ Forwarder = &DefaultForwarder{}

// DefaultForwarder - default 'last mile' webhook Forwarder
type DefaultForwarder struct {
	hClient *http.Client
	rClient *retryablehttp.Client

	logger *zap.SugaredLogger
}

// Opts - configuration
type Opts struct {
	Retries  int
	Insecure bool
	Logger   *zap.SugaredLogger
}

// NewDefaultForwarder - create an instance of default Forwarder
func NewDefaultForwarder(opts *Opts) *DefaultForwarder {

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

	return &DefaultForwarder{rClient: client, hClient: httpClient, logger: opts.Logger}
}

// Forward - relaying incoming webhook to original destination
func (r *DefaultForwarder) Forward(wh types.Event) *types.EventStatus {
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
			"response_body", bodyStr,
			"request_body", wh.Body,
		)
	}

	return &types.EventStatus{
		ID:         wh.Meta.ID,
		StatusCode: resp.StatusCode,
		Message:    bodyStr,
		Retries:    retries,
	}
}
