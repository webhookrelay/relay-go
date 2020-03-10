package client

import (
	"context"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/webhookrelay/relay-go/pkg/cond"
	"github.com/webhookrelay/relay-go/pkg/forward"
	"github.com/webhookrelay/relay-go/pkg/gopool"
	"github.com/webhookrelay/relay-go/pkg/logger"
	"github.com/webhookrelay/relay-go/pkg/types"
)

type WebhookRelayClient interface {
	// start webhook relay
	StartRelay(ctx context.Context, filter *Filter) error
	RelayReady() <-chan bool
}

var _ WebhookRelayClient = &DefaultClient{}

// Filter - optional filter that can be passed to the server
type Filter struct {
	Bucket, Destination string
	Buckets             []string // multiple bucket filtering based on name or ID
}

// concurrency options
var (
	workers = 256
	queue   = 1
)

// Opts - client configuration
type Opts struct {
	HTTPClient              *http.Client
	AccessKey, AccessSecret string
	// Optional way to turn off TLS certificate validation
	InsecureSkipVerify bool
	Forwarder          forward.Forwarder
	Debug              bool
	Logger             *zap.SugaredLogger
	// Websocket server address, defaults to
	// wss://my.webhookrelay.com/
	ServerAddress string
}

// DefaultClient - default client that connects to webhookrelay service via gRPC protocol
type DefaultClient struct {
	httpClient   *http.Client
	forwarder    forward.Forwarder
	wsConn       *websocket.Conn
	wsHealthPing chan *types.Event
	opts         *Opts
	filter       *Filter
	readyCond    *cond.Cond
	goPool       *gopool.Pool
	readyMu      *sync.Mutex
	logger       *zap.SugaredLogger
}

// NewDefaultClient - create new default client with given options
func NewDefaultClient(opts *Opts) *DefaultClient {
	if opts.Logger == nil {
		opts.Logger = logger.GetLoggerInstance(logger.DefaultLogLevel).Sugar()
	}

	if opts.HTTPClient == nil {
		opts.HTTPClient = http.DefaultClient
	}

	return &DefaultClient{
		opts:         opts,
		httpClient:   opts.HTTPClient,
		logger:       opts.Logger,
		forwarder:    opts.Forwarder,
		goPool:       gopool.NewPool(workers, queue, 1),
		readyCond:    &cond.Cond{},
		readyMu:      &sync.Mutex{},
		wsHealthPing: make(chan *types.Event),
	}
}

// StartRelay - starts relay agent
func (c *DefaultClient) StartRelay(ctx context.Context, filter *Filter) error {
	c.filter = filter
	return c.startWebSocketRelay(ctx)
}

// RelayReady - relay notification channel, closed when relay is ready
func (c *DefaultClient) RelayReady() <-chan bool {

	rCh := make(chan bool)

	cCh := make(chan int, 1)

	c.readyCond.Register(cCh, 0)
	go func() {
		<-cCh
		close(rCh)
	}()

	return rCh
}
