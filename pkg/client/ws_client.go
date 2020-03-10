package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
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

	webSocketAddress := c.opts.ServerAddress + "/v1/socket"

	if strings.HasPrefix(webSocketAddress, "https://") {
		webSocketAddress = strings.Replace(webSocketAddress, "https", "wss", 1)
	}
	if strings.HasPrefix(webSocketAddress, "http://") {
		webSocketAddress = strings.Replace(webSocketAddress, "http", "ws", 1)
	}

	if c.opts.InsecureSkipVerify {
		websocket.DefaultDialer.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	conn, _, err := websocket.DefaultDialer.Dial(webSocketAddress, nil)
	if err != nil {
		c.logger.Errorw("websocket connection to Webhook Relay failed",
			"error", err,
			"address", webSocketAddress,
		)
		return nil, fmt.Errorf("websocket dial to '%s' failed, error: %s", webSocketAddress, err)
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
	c.logger.Infof("authenticating to '%s'...", c.opts.ServerAddress)

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
		resp, err := c.forwarder.Forward(event)
		if err != nil {
			return err
		}

		return c.sendResponse(resp)
	default:
		c.logger.Warnf("unknown event type: %s", event, true)
	}
	return nil
}

func (c *DefaultClient) sendResponse(webhookResponse *types.LogUpdateRequest) error {

	bts, err := easyjson.Marshal(webhookResponse)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, c.opts.ServerAddress+"/v1/logs/"+webhookResponse.ID, bytes.NewBuffer(bts))
	if err != nil {
		return err
	}

	req.SetBasicAuth(c.opts.AccessKey, c.opts.AccessSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("unexpected status from Webhook Relay: %d", resp.StatusCode)
		}

		return fmt.Errorf("unexpected status from Webhook Relay: %d (%s)", resp.StatusCode, string(body))
	}

	return nil
}
