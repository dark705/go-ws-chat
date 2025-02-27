package chat

import (
	"context"
	"math/rand/v2"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
)

const (
	HTTPWebSocketEndpoint     = "ws"
	HTTPWebSocketRoutePattern = http.MethodGet + " /" + HTTPWebSocketEndpoint

	maxRandomID                = 10000
	writeChanelBufferSizeBytes = 256
)

type webSocketHandler struct {
	logger         Logger
	wsUpgrader     *websocket.Upgrader
	wsClientConfig ClientConfig
	pubSubHub      PubSubHub
}

func NewWebSocketHandler(logger Logger,
	webSocketUpgrader *websocket.Upgrader,
	wsClientConfig ClientConfig,
	pubSubHub PubSubHub) *webSocketHandler { //nolint:revive
	return &webSocketHandler{
		logger:         logger,
		wsUpgrader:     webSocketUpgrader,
		wsClientConfig: wsClientConfig,
		pubSubHub:      pubSubHub,
	}
}

func (h *webSocketHandler) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	clientID := strconv.Itoa(rand.IntN(maxRandomID)) //nolint:gosec

	wsConnect, err := h.wsUpgrader.Upgrade(responseWriter, request, nil)
	if err != nil {
		h.logError(ctx, request, "chat, webSocketHandler, wsUpgrader.Upgrade", err) // h.wsUpgrader.Upgrade already send http error

		return
	}
	h.logInfo(ctx, request, "chat, webSocketHandler", "new connect, clientID: "+clientID)

	readCh := make(chan []byte)                              // messages FROM ws client
	writeCh := make(chan []byte, writeChanelBufferSizeBytes) // messages TO ws client

	wsClient := &webSocketClient{
		logger:   h.logger,
		config:   h.wsClientConfig,
		clientID: clientID,
		connect:  wsConnect,
		readCh:   readCh,
		writeCh:  writeCh,
	}

	messageHandler := &oneToOneHandler{
		logger:    h.logger,
		pubSubHub: h.pubSubHub,
		clientID:  clientID,
		readCh:    readCh,
		writeCh:   writeCh,
	}

	ctx = context.WithoutCancel(ctx)
	go wsClient.writePump(ctx)
	go wsClient.readPump(ctx)

	ctx, cancel := context.WithCancel(ctx)
	go messageHandler.write(ctx, cancel)
	go messageHandler.read(ctx, cancel)
}

func (h *webSocketHandler) logError(ctx context.Context, _ *http.Request, point string, err error) {
	h.logger.ErrorfContext(ctx, "%s, error: %s", point, err)
}

func (h *webSocketHandler) logInfo(ctx context.Context, _ *http.Request, point, msg string) {
	h.logger.InfofContext(ctx, "%s, msg: %s", point, msg)
}
