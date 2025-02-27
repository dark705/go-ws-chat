package chat

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

var errWrongWSClientMessageType = errors.New("wrong WebSocket client message type")

type ClientConfig struct {
	WriteTimeoutSeconds int
	ReadTimeoutSeconds  int
	ReadLimitPerMessage int
	PingIntervalSeconds int
}

type webSocketClient struct {
	logger   Logger
	config   ClientConfig
	clientID string
	connect  *websocket.Conn
	readCh   chan []byte
	writeCh  chan []byte
}

func (c *webSocketClient) readPump(ctx context.Context) {
	defer func() {
		c.connect.Close()
		close(c.readCh)
		c.logDebug(ctx, "chat, webSocketClient, readPump", "stopped")
	}()
	c.connect.SetReadLimit(int64(c.config.ReadLimitPerMessage))
	c.connect.SetReadDeadline( //nolint:errcheck
		time.Now().Add(time.Duration(c.config.ReadTimeoutSeconds) * time.Second))
	c.connect.SetPongHandler(func(string) error {
		c.logDebug(ctx, "chat, webSocketClient, readPump", "pong")
		c.connect.SetReadDeadline( //nolint:errcheck
			time.Now().Add(time.Duration(c.config.ReadTimeoutSeconds) * time.Second))

		return nil
	})

	for {
		wsMessageType, message, err := c.connect.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logError(ctx, "chat, webSocketClient, readPump, connect.ReadMessage", err)
			}

			break
		}
		c.logDebug(ctx, "chat, webSocketClient, readPump", fmt.Sprintf("received: %s, type: %d", message, wsMessageType))
		if wsMessageType != websocket.TextMessage {
			c.logError(ctx, "chat, webSocketClient, readPump, connect.ReadMessage", errWrongWSClientMessageType)

			break
		}

		c.readCh <- message
	}
}

func (c *webSocketClient) writePump(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(c.config.PingIntervalSeconds) * time.Second)
	defer func() {
		ticker.Stop()
		c.connect.Close()
		c.logDebug(ctx, "chat, webSocketClient, writePump", "stopped")
	}()

	for {
		select {
		case message, ok := <-c.writeCh:
			c.connect.SetWriteDeadline(time.Now().Add(time.Duration(c.config.WriteTimeoutSeconds) * time.Second)) //nolint:errcheck
			if !ok {
				c.connect.WriteMessage(websocket.CloseMessage, []byte{}) //nolint:errcheck

				return
			}

			err := c.connect.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				c.logError(ctx, "chat, webSocketClient, writePump, connect.WriteMessage Text", err)

				return
			}
			c.logDebug(ctx, "chat, webSocketClient, writePump", fmt.Sprintf("sent: %s", message))

		case <-ticker.C:
			c.connect.SetWriteDeadline(time.Now().Add(time.Duration(c.config.WriteTimeoutSeconds) * time.Second)) //nolint:errcheck
			if err := c.connect.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logError(ctx, "chat, webSocketClient, writePump, connect.WriteMessage Ping", err)

				return
			}
			c.logDebug(ctx, "chat, webSocketClient, writePump", "ping")
		}
	}
}

func (c *webSocketClient) logError(ctx context.Context, point string, err error) {
	c.logger.ErrorfContext(ctx, "%s, error: %s", point, err)
}

func (c *webSocketClient) logDebug(ctx context.Context, point, msg string) {
	c.logger.DebugfContext(ctx, "%s, msg: %s", point, msg)
}
