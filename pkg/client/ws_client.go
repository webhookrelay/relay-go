package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mailru/easyjson"
	"github.com/webhookrelay/relay-go/pkg/types"
)

var websocketHealthPingTimeout = time.Second * 53

func (c *DefaultClient) dialWebSocket(ctx context.Context) (*websocket.Conn, error) {

	if c.wsConn != nil {
		// closing any existing connection
		c.wsConn.Close()
	}

	if strings.HasPrefix(c.opts.WebSocketAddress, "https://") {
		c.opts.WebSocketAddress = strings.Replace(c.opts.WebSocketAddress, "https", "wss", 1)
	}
	if strings.HasPrefix(c.opts.WebSocketAddress, "http://") {
		c.opts.WebSocketAddress = strings.Replace(c.opts.WebSocketAddress, "http", "ws", 1)
	}

	if c.opts.InsecureSkipVerify {
		websocket.DefaultDialer.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	conn, _, err := websocket.DefaultDialer.Dial(c.opts.WebSocketAddress, nil)
	if err != nil {
		c.logger.Errorw("websocket connection to Webhook Relay failed",
			"error", err,
			"address", c.opts.WebSocketAddress,
		)
		return nil, fmt.Errorf("websocket dial to '%s' failed, error: %s", c.opts.WebSocketAddress, err)
	}

	return conn, nil
}

func (c *DefaultClient) startWebSocketRelay(ctx context.Context) error {

	wsHealthTimer := time.NewTimer(websocketHealthPingTimeout)

RECONNECT:
	conn, err := c.dialWebSocket(ctx)
	if err != nil {
		// retrying connection forever
		select {
		case <-ctx.Done():
			return nil
		default:
			c.logger.Errorw("websocket connection failed, retrying...",
				"error", err,
			)
			time.Sleep(2 * time.Second)
			goto RECONNECT
		}
	}

	c.wsConn = conn
	defer c.wsConn.Close()

	c.logger.Info("using websocket based transport...")

	readErrCh := make(chan error)

	go func() {
		defer close(readErrCh)
		c.logger.Info("websocket reader process started...")
		defer c.logger.Info("websocket reader process stopped...")
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if !strings.Contains(err.Error(), "closed") {
					c.logger.Errorf("read: %s", err)
					readErrCh <- err
				}

				return
			}
			go func(msg []byte) {
				err = c.handleWSMessage(msg)
				if err != nil {
					c.logger.Errorw("failed to process ws message",
						"error", err,
					)
				}
			}(message)
		}
	}()

	// send authentication message
	c.logger.Infof("authenticating to '%s'...", c.opts.WebSocketAddress)

	bts, err := easyjson.Marshal(&types.ActionRequest{
		Action: "auth",
		Key:    c.opts.AccessKey,
		Secret: c.opts.AccessSecret,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal auth request: %s", err)
	}

	err = conn.WriteMessage(websocket.TextMessage, bts)
	if err != nil {
		c.logger.Errorw("failed to send authentication message",
			"error", err,
		)
	}

	// monitor connection
	for {
		select {
		case <-ctx.Done():
			return nil
		case err, ok := <-readErrCh:
			if ok {
				if err != nil {
					c.logger.Warnf("websocket read failure: %s", err)
					goto RECONNECT
				}

			}

		case <-wsHealthTimer.C:
			c.logger.Info("missing server websocket pings, reconnecting...")
			goto RECONNECT
		case _, ok := <-c.wsHealthPing:
			if !ok {
				return nil
			}
			// reseting the timer
			wsHealthTimer.Reset(websocketHealthPingTimeout)
		}
	}
}

func (c *DefaultClient) handleWSMessage(msg []byte) error {

	var event types.Event
	err := easyjson.Unmarshal(msg, &event)
	if err != nil {
		return fmt.Errorf("failed to unmarshal event: %s", err)
	}

	switch event.Type {
	case "status":
		switch event.Status {
		case "authenticated":
			buckets := c.filter.Buckets
			if c.filter.Bucket != "" {
				buckets = append(buckets, c.filter.Bucket)
			}
			// notifying readiness
			c.readyCond.Notify()

			// subscribing to buckets
			bts, err := easyjson.Marshal(&types.ActionRequest{
				Action:  "subscribe",
				Buckets: buckets,
			})
			if err != nil {
				return fmt.Errorf("failed to marshal subscribe request: %s", err)
			}
			c.logger.Infof("subscribing to buckets: %s", buckets)
			err = c.wsConn.WriteMessage(websocket.TextMessage, bts)
			if err != nil {
				c.logger.Errorw("failed to send subscribe message",
					"error", err,
				)
			}
			return err
		case "unauthorized":
			c.logger.Fatalf("authentication failed, check your credentials")
			return fmt.Errorf("authentication failed")
		case "ping":
			bts, err := easyjson.Marshal(&types.ActionRequest{
				Action: "pong",
			})
			c.wsHealthPing <- &event
			if err != nil {
				return fmt.Errorf("failed to marshal pong request: %s", err)
			}
			err = c.wsConn.WriteMessage(websocket.TextMessage, bts)
			if err != nil {
				c.logger.Errorw("failed to send a message",
					"error", err,
				)
			}
			return err
		}

	case "webhook":
		resp := c.forwarder.Forward(event)
		c.logger.Infow("webhook request relayed",
			"destination", event.Meta.OutputDestination,
			"method", event.Method,
			"bucket", event.Meta.BucketName,
			"status", resp.StatusCode,
			"retries", resp.Retries,
		)
		return nil
	default:
		c.logger.Warnf("unknown event type: %s", event, true)
	}
	return nil
}
