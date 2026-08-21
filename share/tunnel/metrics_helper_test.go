package tunnel

import "testing"

// TestRecordMetricsNilSafe guards against a regression where a *Tunnel
// constructed without metrics (the common case, --metrics unset) would
// panic on the first connection because a nil metrics pointer, once
// wrapped, was mistaken for a non-nil value.
func TestRecordMetricsNilSafe(t *testing.T) {
	tun := &Tunnel{}

	tun.recordTunnelConnection()
	tun.recordTunnelConnectionError()
	tun.recordTunnelActiveConnectionsInc()
	tun.recordTunnelActiveConnectionsDec()
	tun.recordTunnelBytes(10, 20)
	tun.recordTunnelKeepalivePing("success")
}
