package chat

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

var errWrongWSRemoteClientMessageType = errors.New("wrong WebSocket remote client msg type")

type WSClientConfig struct {
	WriteTimeoutSeconds int
	ReadTimeoutSeconds  int
	ReadLimitPerMessage int
	PingIntervalSeconds int
}

type WSClient struct {
	config    WSClientConfig
	logger    Logger
	uniqID    string
	wsConnect *websocket.Conn
	readCh    chan []byte
	writeCh   chan []byte
}

func (c *WSClient) readPump(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
		c.wsConnect.Close()
		close(c.readCh)
	}()
	c.wsConnect.SetReadLimit(int64(c.config.ReadLimitPerMessage))
	c.wsConnect.SetReadDeadline( //nolint:errcheck
		time.Now().Add(time.Duration(c.config.ReadTimeoutSeconds) * time.Second))
	c.wsConnect.SetPongHandler(func(string) error {
		c.logDebug(ctx, "chat, WSClient, readPump", "got pong")
		c.wsConnect.SetReadDeadline( //nolint:errcheck
			time.Now().Add(time.Duration(c.config.ReadTimeoutSeconds) * time.Second))

		return nil
	})

	for {
		messageType, message, err := c.wsConnect.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logError(ctx, "chat, WSClient, readPump, wsConnect.ReadMessage", err)
			}

			break
		}
		c.logDebug(ctx, "chat, WSClient, readPump", fmt.Sprintf("got message type: %d, data: %s", messageType, message))
		if messageType != websocket.TextMessage {
			c.logError(ctx, "chat, WSClient, readPump, wsConnect.ReadMessage", errWrongWSRemoteClientMessageType)

			break
		}

		c.readCh <- message
	}
}

func (c *WSClient) writePump(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	ticker := time.NewTicker(time.Duration(c.config.PingIntervalSeconds) * time.Second)
	defer func() {
		cancel()
		ticker.Stop()
		c.wsConnect.Close()
	}()

	for {
		select {
		case message, ok := <-c.writeCh:
			c.wsConnect.SetWriteDeadline(time.Now().Add(time.Duration(c.config.WriteTimeoutSeconds) * time.Second)) //nolint:errcheck
			if !ok {
				c.wsConnect.WriteMessage(websocket.CloseMessage, []byte{}) //nolint:errcheck

				return
			}

			err := c.wsConnect.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				c.logError(ctx, "chat, WSClient, writePump, wsConnect.WriteMessage Text", err)

				return
			}
			c.logDebug(ctx, "chat, WSClient, writePump", fmt.Sprintf("send message, data: %s", message))

		case <-ticker.C:
			c.wsConnect.SetWriteDeadline(time.Now().Add(time.Duration(c.config.WriteTimeoutSeconds) * time.Second)) //nolint:errcheck
			if err := c.wsConnect.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logError(ctx, "chat, WSClient, writePump, wsConnect.WriteMessage Ping", err)

				return
			}
			c.logDebug(ctx, "chat, WSClient, writePump", "send ping")
		}
	}
}

func (c *WSClient) logError(ctx context.Context, point string, err error) {
	c.logger.ErrorfContext(ctx, "%s, error: %s", point, err)
}

func (c *WSClient) logDebug(ctx context.Context, point, msg string) {
	c.logger.DebugfContext(ctx, "%s, msg: %s", point, msg)
}

func (c *WSClient) logInfo(ctx context.Context, point, msg string) { //nolint:unused
	c.logger.InfofContext(ctx, "%s, msg: %s", point, msg)
}
