package main

import (
	"context"
	"os"
	"os/signal"
	"strings"

	"github.com/webhookrelay/relay-go/pkg/client"
	"github.com/webhookrelay/relay-go/pkg/forward"

	"github.com/heptio/workgroup"
	"go.uber.org/zap"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	EnvRelayKey                  = "RELAY_KEY"
	EnvRelaySecret               = "RELAY_SECRET"
	EnvBuckets                   = "BUCKETS"
	EnvRelayRetries              = "RELAY_RETRIES"
	EnvWebhookRelayServerAddress = "WEBHOOKRELAY_SERVER_ADDRESS"
)

var (
	app = kingpin.New("relayd", "A webhook forwarding client")

	key    = app.Flag("key", "Access token key").OverrideDefaultFromEnvar(EnvRelayKey).Default("").String()
	secret = app.Flag("secret", "Access token secret").OverrideDefaultFromEnvar(EnvRelaySecret).Default("").String()

	debug = app.Flag("debug", "Enabled debugging").OverrideDefaultFromEnvar("DEBUG").Default("false").Bool()

	fwd      = app.Command("forward", "Start forwarding buckets")
	buckets  = fwd.Flag("buckets", "Buckets to forward too").OverrideDefaultFromEnvar(EnvBuckets).Default("").String()
	retries  = fwd.Flag("retries", "Maximum number of retries").OverrideDefaultFromEnvar(EnvRelayRetries).Default("3").Int()
	insecure = fwd.Flag("insecure", "Skip TLS verification when forwarding webhooks").Default("false").Bool()
)

var (
	defaultServerAddress = "https://my.webhookrelay.com:443/v1/socket"
)

func main() {

	ver := "1.0.0"

	kingpin.UsageTemplate(kingpin.CompactUsageTemplate).Version(ver)
	kingpin.CommandLine.Help = "Webhook Relay lightweight client.Learn more on https://webhookrelay.com"
	// kingpin.Parse()

	zapCfg := zap.NewDevelopmentConfig()

	l, _ := zapCfg.Build()
	logger := l.Sugar()

	serverAddress := defaultServerAddress
	if os.Getenv(EnvWebhookRelayServerAddress) != "" {
		serverAddress = os.Getenv(EnvWebhookRelayServerAddress)
	}

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	// Register user
	case fwd.FullCommand():

		if *secret == "" || *key == "" {
			logger.Errorf("--key and --secret flags must be set, alternatively use %s and %s environment variables. To create a token, visit https://my.webhookrelay.com/tokens", EnvRelayKey, EnvRelaySecret)
			os.Exit(1)
		}
		logger.Info("forwarding..")

		forwarder := forward.NewDefaultForwarder(&forward.Opts{
			Retries:  *retries,
			Insecure: *insecure,
			Logger:   logger.With("module", "forwarder"),
		})

		logger.Infof("key: %s secret: %s", *key, *secret)

		c := client.NewDefaultClient(&client.Opts{
			AccessKey:          *key,
			AccessSecret:       *secret,
			InsecureSkipVerify: *insecure,
			Logger:             logger.With("module", "client"),
			Forwarder:          forwarder,
			WebSocketAddress:   serverAddress,
			Debug:              *debug,
		})

		sanitize := func(buckets string) []string {
			parts := strings.Split(buckets, ",")
			for idx, p := range parts {
				parts[idx] = strings.TrimSpace(p)
			}
			return parts
		}

		filter := client.Filter{
			Buckets: sanitize(*buckets),
		}

		// c.StartRelay()
		var g workgroup.Group

		g.Add(func(stop <-chan struct{}) error {
			defer logger.Info("forwarding stopped")

			ctx, cancel := context.WithCancel(context.Background())

			go func() {
				<-stop
				cancel()
				// err := apiServer.Stop()
				// if err != nil {
				// logger.Warnf("failure while stopping API server: %s", err)
				// }
			}()

			err := c.StartRelay(ctx, &filter)
			if err != nil {
				logger.Errorf("failed to start relay client: %s", err)
			}
			return err
		})

		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt)
		g.Add(func(stop <-chan struct{}) error {
			// go func() {
			for range signalChan {
				logger.Info("received an interrupt, shutting down...")
				return nil
			}
			return nil
		})

		err := g.Run()
		if err != nil {
			logger.Errorf("forward exitted with an error: %s", err)
			os.Exit(1)
		}
	}
}
