package forward

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRelayFoward(t *testing.T) {

	if os.Getenv("RELAY_KEY") == "" || os.Getenv("RELAY_SECRET") == "" {
		t.Fatalf("RELAY_KEY or RELAY_SECRET not set")
	}
	relayCLITestBucketName := os.Getenv("BUCKETS")
	testBucketInputURL := os.Getenv("INPUT_URL")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	webhookReceiver := &WebhookServer{}
	go webhookReceiver.Start(":33003")
	time.Sleep(100 * time.Millisecond)
	defer webhookReceiver.Shutdown()

	relayCLI := NewRelayCmd(os.Getenv("RELAY_KEY"), os.Getenv("RELAY_SECRET"))
	go func() {
		// ensure that the bucket has output http://localhost:33003/webhook
		err := relayCLI.Forward(ctx, []string{"--buckets", relayCLITestBucketName})
		if err != nil {
			t.Errorf("failed to start relay process: %s", err)
		}
	}()

	time.Sleep(1 * time.Second)

	var defaultTransport http.RoundTripper = &http.Transport{
		Proxy: nil,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          30,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	httpClient := &http.Client{Transport: defaultTransport}

	t.Run("ForwardWebhook", func(t *testing.T) {

		// wiping server of any webhooks
		defer webhookReceiver.Cleanup()

		method := "POST"
		payload := "foo"

		headers := map[string]string{
			"hdr-1": "foo",
			"hdr-2": "foo",
			"hdr-3": "foo",
			"hdr-4": "foo",
			"hdr-5": "foo",
			"hdr-6": "foo",
		}

		req, err := http.NewRequest(method, testBucketInputURL, bytes.NewBufferString(payload))
		if err != nil {
			t.Fatalf("failed to create req: %s", err)
		}

		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		time.Sleep(1 * time.Second)

		// checking whether we got it
		if len(webhookReceiver.Received()) != 1 {
			t.Errorf("expected to receive 1 webhook, got %d", len(webhookReceiver.Received()))
		} else {

			webhook := webhookReceiver.Received()[0]

			if webhook.Payload != payload {
				t.Errorf("expected payload '%s', got: '%s'", payload, webhook.Payload)
			}
			if webhook.Method != method {
				t.Errorf("expected method '%s', got: '%s'", method, webhook.Method)
			}

			for k, v := range headers {
				if webhook.Headers[strings.Title(k)] != v {
					t.Errorf("expected %s=%s, got: %s=%s", k, v, k, webhook.Headers[k])
				}
			}
		}
	})
}
