package chat

import (
	"context"
	"github.com/gorilla/websocket"
	"math/rand/v2"
	"net/http"
	"strconv"
)

const (
	HTTPWSEndpoint     = "ws"
	HTTPWSRoutePattern = http.MethodGet + " /" + HTTPWSEndpoint
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
	uniqId := strconv.Itoa(rand.IntN(100000))
	wsConnect, err := h.wsUpgrader.Upgrade(responseWriter, request, nil)
	if err != nil {
		h.logError(ctx, request, "chat, wsHandler, wsUpgrader.Upgrade", err) // h.wsUpgrader.Upgrade already send http error

		return
	}
	h.logInfo(ctx, request, "chat, wsHandler", "new connect, uniqID: "+uniqId)

	wsClient := &WSClient{
		config:    h.wsClientConfig,
		logger:    h.logger,
		uniqId:    uniqId,
		wsConnect: wsConnect,
		readCh:    make(chan []byte),
		writeCh:   make(chan []byte, 256),
	}

	ctx = context.Background()
	go wsClient.writePump(ctx)
	go wsClient.readPump(ctx)
	go wsClient.processor(ctx, h.ps)
}

func (h *wsHandler) logError(ctx context.Context, r *http.Request, point string, err error) { //nolint:unused
	h.logger.ErrorfContext(ctx, "%s, error: %s", point, err)
}

func (h *wsHandler) logDebug(ctx context.Context, r *http.Request, point, msg string) { //nolint:unused
	h.logger.DebugfContext(ctx, "%s, msg: %s", point, msg)
}

func (h *wsHandler) logInfo(ctx context.Context, r *http.Request, point, msg string) { //nolint:unused
	h.logger.InfofContext(ctx, "%s, msg: %s", point, msg)
}
