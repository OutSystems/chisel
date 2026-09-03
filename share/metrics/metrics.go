package metrics

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var namespaceRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// Metrics holds all Prometheus metrics for chisel server and client
type Metrics struct {
	namespace string
	registry  *prometheus.Registry
	verbose   bool

	// Server metrics
	ServerAuthAttempts       *prometheus.CounterVec
	ServerSessionSetupDuration prometheus.Histogram
	ServerSessions           *prometheus.CounterVec

	// Client metrics
	ClientConnectionAttempts *prometheus.Counter
	ClientConnectionErrors   *prometheus.CounterVec
	ClientHandshakeDuration  prometheus.Histogram
	ClientConnected          prometheus.Gauge

	// Tunnel metrics (shared by server and client)
	TunnelConnections        *prometheus.Counter
	TunnelConnectionErrors   *prometheus.Counter
	TunnelActiveConnections  prometheus.Gauge
	TunnelBytes              *prometheus.CounterVec
	TunnelKeepalivePings     *prometheus.CounterVec
}

// New creates a new Metrics instance with the given namespace
func New(namespace string) (*Metrics, error) {
	if namespace == "" {
		namespace = "chisel"
	}
	if !namespaceRe.MatchString(namespace) {
		return nil, fmt.Errorf("invalid metrics namespace %q: must match %s", namespace, namespaceRe.String())
	}

	registry := prometheus.NewRegistry()
	m := &Metrics{
		namespace: namespace,
		registry:  registry,
	}

	// Server metrics
	m.ServerAuthAttempts = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "server_auth_attempts_total",
			Help:      "Total SSH authentication attempts, labeled by outcome",
		},
		[]string{"outcome"},
	)

	m.ServerSessionSetupDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "server_session_setup_duration_seconds",
			Help:      "Seconds from WebSocket upgrade to successful config handshake",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5},
		},
	)

	m.ServerSessions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "server_sessions_total",
			Help:      "Total session attempts by terminal outcome",
		},
		[]string{"outcome"},
	)

	// Client metrics
	clientConnectionAttempts := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "client_connection_attempts_total",
			Help:      "Total connection attempts including retries after disconnect",
		},
	)
	m.ClientConnectionAttempts = &clientConnectionAttempts

	m.ClientConnectionErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "client_connection_errors_total",
			Help:      "Total connection failures by category",
		},
		[]string{"cause"},
	)

	m.ClientHandshakeDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "client_handshake_duration_seconds",
			Help:      "Seconds for config round-trip: send SSH config request to receive server reply",
			Buckets:   []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
		},
	)

	m.ClientConnected = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "client_connected",
			Help:      "1 when tunnel is active (after config verified), 0 when disconnected",
		},
	)

	// Tunnel metrics
	tunnelConnections := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "tunnel_connections_total",
			Help:      "Total tunnel connections opened",
		},
	)
	m.TunnelConnections = &tunnelConnections

	tunnelConnectionErrors := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "tunnel_connection_errors_total",
			Help:      "Total tunnel connection failures (SSH OpenChannel + remote dial errors)",
		},
	)
	m.TunnelConnectionErrors = &tunnelConnectionErrors

	m.TunnelActiveConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "tunnel_active_connections",
			Help:      "Number of connections currently piping data",
		},
	)

	m.TunnelBytes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "tunnel_bytes_total",
			Help:      "Total bytes transferred through the tunnel",
		},
		[]string{"direction"},
	)

	m.TunnelKeepalivePings = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "tunnel_keepalive_pings_total",
			Help:      "Total SSH keepalive pings sent, labeled by outcome",
		},
		[]string{"outcome"},
	)

	// Register all metrics
	registry.MustRegister(
		m.ServerAuthAttempts,
		m.ServerSessionSetupDuration,
		m.ServerSessions,
		*m.ClientConnectionAttempts,
		m.ClientConnectionErrors,
		m.ClientHandshakeDuration,
		m.ClientConnected,
		*m.TunnelConnections,
		*m.TunnelConnectionErrors,
		m.TunnelActiveConnections,
		m.TunnelBytes,
		m.TunnelKeepalivePings,
	)

	return m, nil
}

// SetVerbose enables verbose logging of metric changes
func (m *Metrics) SetVerbose(verbose bool) {
	m.verbose = verbose
}

// Start starts the metrics HTTP server on the given address
func (m *Metrics) Start(addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{}))

	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil && err != http.ErrServerClosed {
			// Log error but don't crash the main application
			println("metrics server error:", err.Error())
		}
	}()

	return nil
}

// RecordTunnelConnection increments the tunnel connections counter
func (m *Metrics) RecordTunnelConnection() {
	(*m.TunnelConnections).Inc()
	if m.verbose {
		println("[metrics] tunnel_connections_total incremented")
	}
}

// RecordTunnelConnectionError increments the tunnel connection errors counter
func (m *Metrics) RecordTunnelConnectionError() {
	(*m.TunnelConnectionErrors).Inc()
	if m.verbose {
		println("[metrics] tunnel_connection_errors_total incremented")
	}
}

// RecordTunnelActiveConnectionsInc increments the active connections gauge
func (m *Metrics) RecordTunnelActiveConnectionsInc() {
	m.TunnelActiveConnections.Inc()
	if m.verbose {
		println("[metrics] tunnel_active_connections incremented")
	}
}

// RecordTunnelActiveConnectionsDec decrements the active connections gauge
func (m *Metrics) RecordTunnelActiveConnectionsDec() {
	m.TunnelActiveConnections.Dec()
	if m.verbose {
		println("[metrics] tunnel_active_connections decremented")
	}
}

// RecordTunnelBytes records bytes sent and received
func (m *Metrics) RecordTunnelBytes(sent, received int64) {
	m.TunnelBytes.WithLabelValues("sent").Add(float64(sent))
	m.TunnelBytes.WithLabelValues("received").Add(float64(received))
	if m.verbose {
		println("[metrics] tunnel_bytes_total{direction=\"sent\"} +=", sent)
		println("[metrics] tunnel_bytes_total{direction=\"received\"} +=", received)
	}
}

// RecordTunnelKeepalivePing records a keepalive ping outcome
func (m *Metrics) RecordTunnelKeepalivePing(outcome string) {
	m.TunnelKeepalivePings.WithLabelValues(outcome).Inc()
	if m.verbose {
		println("[metrics] tunnel_keepalive_pings_total{outcome=\"" + outcome + "\"} incremented")
	}
}
