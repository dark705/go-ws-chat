package chat

import (
	"context"
	"math/rand/v2"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
)

const (
	HTTPWSEndpoint     = "ws"
	HTTPWSRoutePattern = http.MethodGet + " /" + HTTPWSEndpoint

	maxRandomID                = 10000
	writeChanelBufferSizeBytes = 256
)

type wsHandler struct {
	logger         Logger
	wsUpgrader     *websocket.Upgrader
	wsClientConfig WSClientConfig
	ps             PubSub
}

func NewWSHandler(logger Logger, webSocketUpgrader *websocket.Upgrader, wsClientConfig WSClientConfig, ps PubSub) *wsHandler { //nolint:revive
	return &wsHandler{
		logger:         logger,
		wsUpgrader:     webSocketUpgrader,
		wsClientConfig: wsClientConfig,
		ps:             ps,
	}
}

func (h *wsHandler) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	uniqID := strconv.Itoa(rand.IntN(maxRandomID)) //nolint:gosec
	wsConnect, err := h.wsUpgrader.Upgrade(responseWriter, request, nil)
	if err != nil {
		h.logError(ctx, request, "chat, wsHandler, wsUpgrader.Upgrade", err) // h.wsUpgrader.Upgrade already send http error

		return
	}
	h.logInfo(ctx, request, "chat, wsHandler", "new connect, uniqID: "+uniqID)

	wsClient := &WSClient{
		config:    h.wsClientConfig,
		logger:    h.logger,
		uniqID:    uniqID,
		wsConnect: wsConnect,
		readCh:    make(chan []byte),
		writeCh:   make(chan []byte, 2566),
	}

	ctx = context.WithoutCancel(ctx)
	go wsClient.writePump(ctx)
	go wsClient.readPump(ctx)
	go wsClient.processor(ctx, h.ps)
}

func (h *wsHandler) logError(ctx context.Context, _ *http.Request, point string, err error) {
	h.logger.ErrorfContext(ctx, "%s, error: %s", point, err)
}

func (h *wsHandler) logDebug(ctx context.Context, _ *http.Request, point, msg string) { //nolint:unused
	h.logger.DebugfContext(ctx, "%s, msg: %s", point, msg)
}

func (h *wsHandler) logInfo(ctx context.Context, _ *http.Request, point, msg string) {
	h.logger.InfofContext(ctx, "%s, msg: %s", point, msg)
}
