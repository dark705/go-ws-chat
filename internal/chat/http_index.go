package chat

import (
	"context"
	"html/template"
	"log"
	"net/http"
)

const (
	HTTPIndexRoutePattern = http.MethodGet + " /"
)

type httpIndexHandler struct {
	logger Logger
	tpl    *template.Template
}

func NewHTTPIndexHandler(logger Logger) *httpIndexHandler { //nolint:revive
	tpl, err := template.ParseFiles("./web/template/index.html")
	failOnError(err, "fail parse template: ./web/template/index.html")

	return &httpIndexHandler{
		logger: logger,
		tpl:    tpl,
	}
}

func (h *httpIndexHandler) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	err := h.tpl.Execute(responseWriter, struct {
		WSUrl               string
		MessageTypeSettings messageType
		MessageTypeText     messageType
	}{
		WSUrl:               HTTPWebSocketEndpoint,
		MessageTypeSettings: messageTypeSettings,
		MessageTypeText:     messageTypeText,
	})
	if err != nil {
		h.logError(ctx, request, "chat, httpIndexHandler, tpl.Execute", err)

		return
	}
}

func (h *httpIndexHandler) logError(ctx context.Context, _ *http.Request, point string, err error) {
	h.logger.ErrorfContext(ctx, "%s, error: %s", point, err)
}

func failOnError(err error, message string) {
	if err != nil {
		log.Fatalf("%s: error: %s", message, err)
	}
}
