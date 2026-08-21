package tunnel

func (t *Tunnel) recordTunnelConnection() {
	if t.metrics == nil {
		return
	}
	t.metrics.RecordTunnelConnection()
}

func (t *Tunnel) recordTunnelConnectionError() {
	if t.metrics == nil {
		return
	}
	t.metrics.RecordTunnelConnectionError()
}

func (t *Tunnel) recordTunnelActiveConnectionsInc() {
	if t.metrics == nil {
		return
	}
	t.metrics.RecordTunnelActiveConnectionsInc()
}

func (t *Tunnel) recordTunnelActiveConnectionsDec() {
	if t.metrics == nil {
		return
	}
	t.metrics.RecordTunnelActiveConnectionsDec()
}

func (t *Tunnel) recordTunnelBytes(sent, received int64) {
	if t.metrics == nil {
		return
	}
	t.metrics.RecordTunnelBytes(sent, received)
}

func (t *Tunnel) recordTunnelKeepalivePing(outcome string) {
	if t.metrics == nil {
		return
	}
	t.metrics.RecordTunnelKeepalivePing(outcome)
}
