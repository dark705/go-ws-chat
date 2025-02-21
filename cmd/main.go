package main

import (
	"github.com/dark705/go-ws-chat/internal/pubsub"
	"github.com/gorilla/websocket"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/dark705/go-ws-chat/internal/chat"
	"github.com/dark705/go-ws-chat/internal/config"
	"github.com/dark705/go-ws-chat/internal/httpserver"
	"github.com/dark705/go-ws-chat/internal/kuberprobe"
	"github.com/dark705/go-ws-chat/internal/prometheus"
	"github.com/dark705/go-ws-chat/internal/slog"
	promhttpmetrics "github.com/slok/go-http-metrics/metrics/prometheus"
	promhttpmiddleware "github.com/slok/go-http-metrics/middleware"
	promhttpmiddlewarestd "github.com/slok/go-http-metrics/middleware/std"
)

func main() {
	envConfig := config.GetConfigFromEnv()

	logger := slog.New(slog.Config{Level: envConfig.LogLevel})
	logger.Infof("app, version: %s", envConfig.Version)

	prometheusServer := prometheus.NewServer(prometheus.Config{HTTPListenPort: envConfig.PrometheusPort}, logger)
	prometheusServer.Run()
	defer prometheusServer.Stop()

	wsUpgrader := &websocket.Upgrader{
		ReadBufferSize:  envConfig.WebSocketUpgraderReadBufferSize,
		WriteBufferSize: envConfig.WebSocketUpgraderWriteBufferSize,
	}
	if !envConfig.WebSocketUpgraderCheckOrigin {
		wsUpgrader.CheckOrigin = func(r *http.Request) bool { return true }
	}

	//ps := pubsub.NewRndEcho(logger)
	ps := pubsub.NewInmemory(logger)

	chatWSHandler := chat.NewWSHandler(logger, wsUpgrader, chat.WSClientConfig{
		WriteTimeoutSeconds: envConfig.WebSocketHandlerWriteTimeoutSeconds,
		ReadTimeoutSeconds:  envConfig.WebSocketHandlerReadTimeoutSeconds,
		ReadLimitPerMessage: envConfig.WebSocketHandlerReadLimitPerMessage,
		PingIntervalSeconds: envConfig.WebSocketHandlerPingIntervalSeconds,
	}, ps)

	chatHTTPIndexHandler := chat.NewHTTPIndexHandler(logger)
	httpKuberProbeHandler := kuberprobe.NewHTTPHandler(logger,
		envConfig.KuberProbeStartupSeconds,
		envConfig.KuberProbeProbabilityLive,
		envConfig.KuberProbeProbabilityReady)

	httpHandler := http.NewServeMux()
	httpHandler.Handle(chat.HTTPWSRoutePattern, chatWSHandler)
	httpHandler.Handle(chat.HTTPIndexRoutePattern, chatHTTPIndexHandler)
	httpHandler.Handle(kuberprobe.HTTPRoutePattern, httpKuberProbeHandler)

	prometheusMiddlewareHandler := promhttpmiddleware.New(promhttpmiddleware.Config{
		Recorder: prometheus.NewFilterRecorder(
			promhttpmetrics.NewRecorder(promhttpmetrics.Config{}), []string{}),
	})

	httpHandlerWithMetric := promhttpmiddlewarestd.Handler("", prometheusMiddlewareHandler, httpHandler)

	httpServer := httpserver.NewServer(httpserver.Config{
		Name:                          "go-ws-chat",
		HTTPListenPort:                envConfig.HTTPPort,
		RequestHeaderMaxBytes:         envConfig.HTTPRequestHeaderMaxSize,
		ReadHeaderTimeoutMilliseconds: envConfig.HTTPRequestReadHeaderTimeoutMilliseconds,
	}, logger, httpHandlerWithMetric)

	httpServer.Run()
	defer httpServer.Stop()

	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, syscall.SIGINT, syscall.SIGTERM)

	logger.Infof("got signal from OS: %v. shutdown...", <-osSignals)
}
