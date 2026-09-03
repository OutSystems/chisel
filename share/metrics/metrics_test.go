package metrics

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNewDefaultsNamespace(t *testing.T) {
	m, err := New("")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if m.namespace != "chisel" {
		t.Fatalf("expected default namespace 'chisel', got %q", m.namespace)
	}
}

func TestNewCustomNamespace(t *testing.T) {
	m, err := New("outsystemscc")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if m.namespace != "outsystemscc" {
		t.Fatalf("expected namespace 'outsystemscc', got %q", m.namespace)
	}
}

func TestNewInvalidNamespace(t *testing.T) {
	for _, ns := range []string{"1bad", "bad-name", "bad.name", "bad name"} {
		if _, err := New(ns); err == nil {
			t.Fatalf("expected error for invalid namespace %q, got nil", ns)
		}
	}
}

func TestRecordTunnelConnection(t *testing.T) {
	m, err := New("chisel")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	m.RecordTunnelConnection()
	m.RecordTunnelConnection()

	mfs, gatherErr := m.registry.Gather()
	if gatherErr != nil {
		t.Fatalf("gather failed: %s", gatherErr)
	}
	var found bool
	for _, mf := range mfs {
		if mf.GetName() == "chisel_tunnel_connections_total" {
			found = true
			if got := mf.GetMetric()[0].GetCounter().GetValue(); got != 2 {
				t.Fatalf("expected counter value 2, got %v", got)
			}
		}
	}
	if !found {
		t.Fatal("chisel_tunnel_connections_total not found in registry")
	}
}

func TestRecordTunnelBytes(t *testing.T) {
	m, err := New("chisel")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	m.RecordTunnelBytes(100, 50)

	if got := testutilGetCounterValue(t, m, "chisel_tunnel_bytes_total", "direction", "sent"); got != 100 {
		t.Fatalf("expected sent=100, got %v", got)
	}
	if got := testutilGetCounterValue(t, m, "chisel_tunnel_bytes_total", "direction", "received"); got != 50 {
		t.Fatalf("expected received=50, got %v", got)
	}
}

func TestRecordTunnelActiveConnections(t *testing.T) {
	m, err := New("chisel")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	m.RecordTunnelActiveConnectionsInc()
	m.RecordTunnelActiveConnectionsInc()
	m.RecordTunnelActiveConnectionsDec()

	mfs, _ := m.registry.Gather()
	for _, mf := range mfs {
		if mf.GetName() == "chisel_tunnel_active_connections" {
			if got := mf.GetMetric()[0].GetGauge().GetValue(); got != 1 {
				t.Fatalf("expected gauge value 1, got %v", got)
			}
			return
		}
	}
	t.Fatal("chisel_tunnel_active_connections not found")
}

func TestRecordTunnelKeepalivePing(t *testing.T) {
	m, err := New("chisel")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	m.RecordTunnelKeepalivePing("success")
	m.RecordTunnelKeepalivePing("timeout")
	m.RecordTunnelKeepalivePing("success")

	if got := testutilGetCounterValue(t, m, "chisel_tunnel_keepalive_pings_total", "outcome", "success"); got != 2 {
		t.Fatalf("expected success=2, got %v", got)
	}
	if got := testutilGetCounterValue(t, m, "chisel_tunnel_keepalive_pings_total", "outcome", "timeout"); got != 1 {
		t.Fatalf("expected timeout=1, got %v", got)
	}
}

func TestStartServesMetricsEndpoint(t *testing.T) {
	m, err := New("chisel")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	m.RecordTunnelConnection()

	addr := "127.0.0.1:19191"
	if err := m.Start(addr); err != nil {
		t.Fatalf("Start failed: %s", err)
	}
	// give the goroutine a moment to bind the listener
	var resp *http.Response
	for i := 0; i < 20; i++ {
		resp, err = http.Get("http://" + addr + "/metrics")
		if err == nil {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("GET /metrics failed: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "chisel_tunnel_connections_total 1") {
		t.Fatalf("expected metric in body, got: %s", body)
	}
}

// testutilGetCounterValue extracts a CounterVec's value for a single label pair.
func testutilGetCounterValue(t *testing.T, m *Metrics, name, labelName, labelValue string) float64 {
	t.Helper()
	mfs, err := m.registry.Gather()
	if err != nil {
		t.Fatalf("gather failed: %s", err)
	}
	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, metric := range mf.GetMetric() {
			for _, l := range metric.GetLabel() {
				if l.GetName() == labelName && l.GetValue() == labelValue {
					return metric.GetCounter().GetValue()
				}
			}
		}
	}
	t.Fatalf("metric %s{%s=%q} not found", name, labelName, labelValue)
	return 0
}
