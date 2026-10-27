package tunnel

import (
	"bytes"
	"testing"

	"github.com/jpillora/chisel/share/cio"
	"golang.org/x/crypto/ssh"
)

type rejectedChannel struct {
	extra   string
	reason  ssh.RejectionReason
	message string
}

func (c *rejectedChannel) Accept() (ssh.Channel, <-chan *ssh.Request, error) {
	panic("channel should not be accepted")
}

func (c *rejectedChannel) Reject(reason ssh.RejectionReason, message string) error {
	c.reason, c.message = reason, message
	return nil
}

func (c *rejectedChannel) ChannelType() string { return "chisel" }
func (c *rejectedChannel) ExtraData() []byte   { return []byte(c.extra) }

func TestDeniedOutboundConnectionLoggedAtInfo(t *testing.T) {
	var logs bytes.Buffer
	logger := cio.NewLoggerFlag("server", 0)
	logger.Info = true
	logger.SetOutput(&logs)
	tun := &Tunnel{Config: Config{Logger: logger, Outbound: false}}

	ch := &rejectedChannel{extra: "example.com:80\nserver: forged"}
	tun.handleSSHChannel(nil, ch)

	if ch.reason != ssh.Prohibited || ch.message != "Denied outbound connection" {
		t.Fatalf("unexpected rejection: %v %q", ch.reason, ch.message)
	}
	const want = "server: Denied outbound connection to \"example.com:80\\nserver: forged\"\n"
	if got := logs.String(); got != want {
		t.Fatalf("unexpected log:\n got: %q\nwant: %q", got, want)
	}
}
