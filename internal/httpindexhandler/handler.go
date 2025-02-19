package httpindexhandler

import (
	"context"
	"html/template"
	"log"
	"net/http"
)

type Logger interface {
	Debugf(format string, args ...any)
	DebugfContext(ctx context.Context, format string, args ...any)

	Infof(format string, args ...any)
	InfofContext(ctx context.Context, format string, args ...any)

	Warnf(format string, args ...any)
	WarnfContext(ctx context.Context, format string, args ...any)

	Errorf(format string, args ...any)
	ErrorfContext(ctx context.Context, format string, args ...any)
}

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
		WSUrl string
	}{WSUrl: "ws"})
	if err != nil {
		h.logError(ctx, responseWriter, request, err, "httpIndexHandler, tpl.Execute")

		return
	}
}

func (h *httpIndexHandler) handleError(ctx context.Context, w http.ResponseWriter, r *http.Request, err error, point string) { //nolint:unused
	h.logError(ctx, w, r, err, point)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (h *httpIndexHandler) logError(ctx context.Context, w http.ResponseWriter, r *http.Request, err error, point string) {
	h.logger.ErrorfContext(ctx, "%s, error: %s", point, err)
}

func failOnError(err error, message string) {
	if err != nil {
		log.Fatalf("%s: error: %s", message, err)
	}
}
