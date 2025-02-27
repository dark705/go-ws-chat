package chat

import (
	"context"
	"encoding/json"
	"errors"
)

var errFailWriteToClientChan = errors.New("fail write to client channel")

type messageType int

const (
	messageTypeSettings messageType = iota
	messageTypeText
)

type Message struct {
	Typ messageType `json:"type"`
}

type SettingsMessage struct {
	Message
	ID string `json:"clientID"`
}

type TextMessageWrite struct {
	Message
	Text string `json:"text"`
}

type TextMessageRead struct {
	Text string `json:"text"`
	To   string `json:"to"`
}

type PubSubHub interface {
	Sub(ctx context.Context, id string) (chan string, error)
	Pub(ctx context.Context, id, message string) error
}

type oneToOneHandler struct {
	logger    Logger
	pubSubHub PubSubHub
	clientID  string
	readCh    chan []byte
	writeCh   chan []byte
}

func (h *oneToOneHandler) read(ctx context.Context, cancel context.CancelFunc) {
	defer func() {
		cancel()
		h.logDebug(ctx, "chat, oneToOneHandler, read", "stopped")
	}()

	for message := range h.readCh {
		var textMessageRead TextMessageRead
		err := json.Unmarshal(message, &textMessageRead)
		if err != nil {
			h.logError(ctx, "chat, oneToOneHandler, read, json.Unmarshal", err)
		}

		err = h.pubSubHub.Pub(ctx, textMessageRead.To, textMessageRead.Text)
		if err != nil {
			h.logError(ctx, "chat, oneToOneHandler, read, pubSubHub.Pub", err)
		}
	}
}

func (h *oneToOneHandler) write(ctx context.Context, cancel context.CancelFunc) {
	defer func() {
		cancel()
		close(h.writeCh)
		h.logDebug(ctx, "chat, oneToOneHandler, write", "stopped")
	}()

	var settingsMessage SettingsMessage
	settingsMessage.Typ = messageTypeSettings
	settingsMessage.ID = h.clientID
	message, err := json.Marshal(settingsMessage)
	if err != nil {
		h.logError(ctx, "chat, oneToOneHandler, write, json.Marshal(settingsMessage)", err)

		return
	}
	h.writeCh <- message

	subCh, err := h.pubSubHub.Sub(ctx, h.clientID)
	if err != nil {
		h.logError(ctx, "chat, oneToOneHandler, write, pubSubHub.Sub", err)

		return
	}

	for subMessage := range subCh {
		var textMessageWrite TextMessageWrite
		textMessageWrite.Typ = messageTypeText
		textMessageWrite.Text = subMessage

		message, err = json.Marshal(textMessageWrite)
		if err != nil {
			h.logError(ctx, "chat, oneToOneHandler, write, json.Marshal(textMessageWrite)", err)

			return
		}

		select {
		case h.writeCh <- message:
		default:
			h.logError(ctx, "chat, oneToOneHandler, write, default", errFailWriteToClientChan)

			return
		}
	}
}

func (h *oneToOneHandler) logError(ctx context.Context, point string, err error) {
	h.logger.ErrorfContext(ctx, "%s, error: %s", point, err)
}

func (h *oneToOneHandler) logDebug(ctx context.Context, point string, msg string) {
	h.logger.DebugfContext(ctx, "%s, msg: %s", point, msg)
}
