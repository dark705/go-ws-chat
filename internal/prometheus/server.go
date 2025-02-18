package prometheus

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	Fatalf(format string, args ...any)
	FatalfContext(ctx context.Context, format string, args ...any)
}

const (
	shutdownMaxTimeout = 5 * time.Second
	readHeaderTimeout  = 2000 * time.Millisecond
)

type Server struct {
	httpServer *http.Server
	logger     Logger
	config     Config
}

type Config struct {
	HTTPListenIP   string
	HTTPListenPort string
}

func NewServer(config Config, logger Logger, metrics ...prometheus.Collector) *Server {
	for _, metric := range metrics {
		prometheus.MustRegister(metric)
	}

	return &Server{
		logger:     logger,
		config:     config,
		httpServer: &http.Server{Handler: promhttp.Handler(), ReadHeaderTimeout: readHeaderTimeout},
	}
}

func (s *Server) Run() {
	address := s.config.HTTPListenIP + ":" + s.config.HTTPListenPort
	s.logger.Infof("prometheusServer, start on: %s", address)
	listener, err := net.Listen("tcp", address)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		failOnError(err, "prometheusServer, fail open port")
	}
	go func() {
		err := s.httpServer.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			failOnError(err, "prometheusServer, fail start")
		}
	}()
}

func (s *Server) Stop() {
	s.logger.Infof("prometheusServer, stop...")
	ctx, cancel := context.WithTimeout(context.Background(), shutdownMaxTimeout)
	err := s.httpServer.Shutdown(ctx)
	if err != nil {
		s.logger.Errorf("prometheusServer, fail stop")
		cancel()

		return
	}
	s.logger.Infof("prometheusServer, success stop")
	cancel()
}

func failOnError(err error, message string) {
	if err != nil {
		log.Fatalf("%s: %s", message, err)
	}
}
