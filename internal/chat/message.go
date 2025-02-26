package chat

import (
	"context"
	"encoding/json"
)

type MessageType int

const (
	MessageTypeSettings MessageType = iota
	MessageTypeText
)

type Message struct {
	Typ MessageType `json:"type"`
}

type SettingsMessage struct {
	Message
	ID string `json:"uniqID"`
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

type MessageHandler struct {
	readCh    chan []byte
	writeCh   chan []byte
	uniqID    string
	PubSubHub PubSubHub
	logger    Logger
}

func (h *MessageHandler) Process(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	// read
	go func() {
		defer cancel()
		for message := range h.readCh {
			var textMessageRead TextMessageRead
			err := json.Unmarshal(message, &textMessageRead)
			if err != nil {
				h.logError(ctx, "chat, WSClient, processor, json.Unmarshal", err)
			}

			err = h.PubSubHub.Pub(ctx, textMessageRead.To, textMessageRead.Text)
			if err != nil {
				h.logError(ctx, "chat, WSClient, processor, pubSub.Pub", err)
			}
		}
	}()

	// write
	go func() {
		defer func() {
			cancel()
			close(h.writeCh)
		}()

		var settingsMessage SettingsMessage
		settingsMessage.Typ = MessageTypeSettings
		settingsMessage.ID = h.uniqID
		message, err := json.Marshal(settingsMessage)
		if err != nil {
			h.logError(ctx, "chat, WSClient, processor, json.Marshal(settingsMessage)", err)

			return
		}
		h.writeCh <- message
		subCh, err := h.PubSubHub.Sub(ctx, h.uniqID)
		if err != nil {
			h.logError(ctx, "chat, WSClient, processor, pubSub.Sub", err)

			return
		}

		for subMessage := range subCh {
			var textMessageWrite TextMessageWrite
			textMessageWrite.Typ = MessageTypeText
			textMessageWrite.Text = subMessage

			message, err = json.Marshal(textMessageWrite)
			if err != nil {
				h.logError(ctx, "chat, processor, json.Marshal(textMessageWrite)", err)

				return
			}
			select {
			case h.writeCh <- message:
			default:

				return
			}
		}
	}()
}

func (h *MessageHandler) logError(ctx context.Context, point string, err error) {
	h.logger.ErrorfContext(ctx, "%s, error: %s", point, err)
}
