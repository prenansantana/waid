// Package nats provides embedded and external NATS messaging for WAID.
package nats

import (
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"

	"github.com/prenansantana/waid/internal/config"
)

// NATS wraps a NATS client connection and optionally an embedded server.
type NATS struct {
	conn           *nats.Conn
	embeddedServer *server.Server
	logger         *slog.Logger
}

// NewNATS creates a new NATS instance based on the provided configuration.
// If cfg.Embedded is true and cfg.URL is empty, an embedded NATS server is
// started on a random port and the client connects to it. Otherwise the client
// connects to cfg.URL directly.
func NewNATS(cfg config.NATSConfig, logger *slog.Logger) (*NATS, error) {
	n := &NATS{logger: logger}

	if cfg.Embedded && cfg.URL == "" {
		if err := n.startEmbedded(); err != nil {
			return nil, fmt.Errorf("starting embedded NATS: %w", err)
		}
	} else {
		url := cfg.URL
		if url == "" {
			url = nats.DefaultURL
		}
		nc, err := nats.Connect(url,
			nats.RetryOnFailedConnect(true),
			nats.MaxReconnects(5),
			nats.ReconnectWait(500*time.Millisecond),
		)
		if err != nil {
			return nil, fmt.Errorf("connecting to NATS at %s: %w", url, err)
		}
		n.conn = nc
		logger.Info("connected to external NATS", "url", url)
	}

	return n, nil
}

// startEmbedded launches an in-process NATS server on a random available port
// and connects the client to it.
func (n *NATS) startEmbedded() error {
	port, err := randomPort()
	if err != nil {
		return fmt.Errorf("finding free port: %w", err)
	}

	opts := &server.Options{
		Port:      port,
		JetStream: true,
		NoLog:     true,
		NoSigs:    true,
	}

	srv, err := server.NewServer(opts)
	if err != nil {
		return fmt.Errorf("creating embedded server: %w", err)
	}

	srv.Start()

	if !srv.ReadyForConnections(5 * time.Second) {
		srv.Shutdown()
		return fmt.Errorf("embedded NATS server did not become ready in time")
	}

	n.embeddedServer = srv

	url := srv.ClientURL()
	nc, err := nats.Connect(url,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(5),
		nats.ReconnectWait(100*time.Millisecond),
	)
	if err != nil {
		srv.Shutdown()
		return fmt.Errorf("connecting client to embedded NATS: %w", err)
	}

	n.conn = nc
	n.logger.Info("embedded NATS server started", "url", url, "port", port)
	return nil
}

// Publish sends data to the given subject.
// Standard subjects used by WAID:
//
//	waid.inbound.{source}      – inbound messages from a provider
//	waid.identity.{event_type} – identity lifecycle events
func (n *NATS) Publish(subject string, data []byte) error {
	if err := n.conn.Publish(subject, data); err != nil {
		return fmt.Errorf("publishing to %s: %w", subject, err)
	}
	return nil
}

// Subscribe registers a message handler for the given subject.
// The returned subscription can be used to drain or unsubscribe.
func (n *NATS) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	sub, err := n.conn.Subscribe(subject, handler)
	if err != nil {
		return nil, fmt.Errorf("subscribing to %s: %w", subject, err)
	}
	return sub, nil
}

// Ping checks whether the client connection is still live.
// It returns an error if the connection is closed or draining.
func (n *NATS) Ping() error {
	if n.conn == nil || n.conn.IsClosed() || n.conn.IsDraining() {
		return fmt.Errorf("NATS connection is not active")
	}
	return n.conn.FlushTimeout(2 * time.Second)
}

// Close drains the client connection and shuts down the embedded server if one
// was started. It is safe to call Close more than once.
func (n *NATS) Close() error {
	if n.conn != nil && !n.conn.IsClosed() {
		if err := n.conn.Drain(); err != nil {
			n.logger.Warn("draining NATS connection", "err", err)
		}
	}
	if n.embeddedServer != nil {
		n.embeddedServer.Shutdown()
		n.embeddedServer.WaitForShutdown()
		n.logger.Info("embedded NATS server stopped")
	}
	return nil
}

// randomPort asks the OS for an available TCP port and returns it.
func randomPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port, nil
}
