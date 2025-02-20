package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"time"
)

var errWrongWSRemoteClientMessageType = errors.New("wrong WebSocket remote client msg type")

const (
	ClientDataTypeSettings = iota
	ClientDataTypeMessage
)

type ClientData struct {
	Typ int `json:"type"`
}

type ClientDataSettings struct {
	ClientData
	ID string `json:"uniqId"`
}

type ClientDataMessageTo struct {
	ClientData
	Message string `json:"message"`
}

type ClientDataMessageFrom struct {
	Message string `json:"message"`
	To      string `json:"to"`
}

type WSClientConfig struct {
	WriteTimeoutSeconds int
	ReadTimeoutSeconds  int
	ReadLimitPerMessage int
	PingIntervalSeconds int
}

type WSClient struct {
	config    WSClientConfig
	logger    Logger
	uniqId    string
	wsConnect *websocket.Conn
	readCh    chan []byte
	writeCh   chan []byte
}

func (c *WSClient) readPump(ctx context.Context) {
	defer func() {
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
		mt, message, err := c.wsConnect.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logError(ctx, "chat, WSClient, readPump, wsConnect.ReadMessage", err)
			}

			break
		}
		c.logDebug(ctx, "chat, WSClient, readPump", fmt.Sprintf("got message, type: %d, data: %s", mt, message))
		if mt != websocket.TextMessage {
			c.logError(ctx, "chat, WSClient, readPump, wsConnect.ReadMessage", errWrongWSRemoteClientMessageType)

			break
		}

		c.readCh <- message
	}
}

func (c *WSClient) writePump(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(c.config.PingIntervalSeconds) * time.Second)
	defer func() {
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

func (c *WSClient) writeSettings(ctx context.Context) {
	var settingsWSData ClientDataSettings
	settingsWSData.Typ = ClientDataTypeSettings
	settingsWSData.ID = c.uniqId
	m, err := json.Marshal(settingsWSData)
	if err != nil {
		c.logError(ctx, "chat, WSClient, writeSettings, json.Marshal", err)

		c.wsConnect.Close()
	}
	fmt.Println(string(m))
	c.writeCh <- m
}

func (c *WSClient) echoTest(ctx context.Context) {
	defer func() {
		close(c.writeCh)
		c.wsConnect.Close()
	}()
	for {
		m := <-c.readCh
		var from ClientDataMessageFrom
		err := json.Unmarshal(m, &from)
		if err != nil {
			c.logError(ctx, "chat, WSClient, echoTest, json.Unmarshal", err)

			return
		}

		var to ClientDataMessageTo
		to.Typ = ClientDataTypeMessage
		to.Message = from.Message

		m, err = json.Marshal(to)
		if err != nil {
			c.logError(ctx, "chat, WSClient, echoTest, json.Marshal", err)

			return
		}
		c.writeCh <- m
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
