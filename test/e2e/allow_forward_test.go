package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	chclient "github.com/jpillora/chisel/client"
	chserver "github.com/jpillora/chisel/server"
	"github.com/jpillora/chisel/share/settings"
	"golang.org/x/crypto/ssh"
)

const forwardDisabledMsg = "Forward port forwarding not enabled on server"

func startForwardTestServer(t *testing.T, config *chserver.Config) string {
	t.Helper()
	s, err := chserver.NewServer(config)
	if err != nil {
		t.Fatal(err)
	}
	s.Debug = debug
	port := availablePort()
	if err := s.Start("127.0.0.1", port); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return "127.0.0.1:" + port
}

// sendConfigResult is sendConfig without the failure on a rejected config.
func sendConfigResult(t *testing.T, sc ssh.Conn, remotes []*settings.Remote) (bool, string) {
	t.Helper()
	cfg, err := json.Marshal(settings.Config{Version: "0", Remotes: remotes})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	ok, reply, err := sc.SendRequest("config", true, cfg)
	if err != nil {
		t.Fatalf("config request: %v", err)
	}
	return ok, string(reply)
}

// forwardRemoteSpecs returns one of each remote kind. Reverse remotes
// need a free port on the server; forward remotes never bind on it.
// Reverse UDP is covered by TestReverseUDP.
func forwardRemoteSpecs() []struct {
	name    string
	remote  string
	reverse bool
} {
	return []struct {
		name    string
		remote  string
		reverse bool
	}{
		{"forward tcp", "0.0.0.0:" + availablePort() + ":127.0.0.1:80", false},
		{"forward socks", "127.0.0.1:" + availablePort() + ":socks", false},
		{"forward udp", "127.0.0.1:" + availableUDPPort() + ":127.0.0.1:53/udp", false},
		{"stdio", "stdio:127.0.0.1:22", false},
		{"reverse tcp", "R:127.0.0.1:" + availablePort() + ":127.0.0.1:80", true},
		{"reverse socks", "R:127.0.0.1:" + availablePort() + ":socks", true},
	}
}

// TestAllowForwardRemoteValidation checks which remotes the server accepts
// in the config request, with and without --allow-forward.
func TestAllowForwardRemoteValidation(t *testing.T) {
	for _, allowForward := range []bool{false, true} {
		t.Run(fmt.Sprintf("allowForward=%v", allowForward), func(t *testing.T) {
			addr := startForwardTestServer(t, &chserver.Config{
				KeySeed: fmt.Sprintf("allow-forward-%v", allowForward),
				Reverse: true,
				Socks5:  true,
				Forward: allowForward,
			})
			for _, spec := range forwardRemoteSpecs() {
				t.Run(spec.name, func(t *testing.T) {
					r, err := settings.DecodeRemote(spec.remote)
					if err != nil {
						t.Fatal(err)
					}
					sc, _, _ := dialChiselSSH(t, addr, "", "")
					defer sc.Close()
					ok, reply := sendConfigResult(t, sc, []*settings.Remote{r})
					wantOK := spec.reverse || allowForward
					if ok != wantOK {
						t.Fatalf("config accepted = %v, want %v (reply %q)", ok, wantOK, reply)
					}
					if !wantOK && !strings.Contains(reply, forwardDisabledMsg) {
						t.Fatalf("unexpected rejection reason: %q", reply)
					}
				})
			}
		})
	}
}

// TestForwardRemoteRejectedByClient runs a real client with a forward remote
// against a server without --allow-forward.
func TestForwardRemoteRejectedByClient(t *testing.T) {
	addr := startForwardTestServer(t, &chserver.Config{KeySeed: "allow-forward-client", Reverse: true})
	var logs syncBuffer
	c, err := chclient.NewClient(&chclient.Config{
		Server:        "http://" + addr,
		Remotes:       []string{availablePort() + ":127.0.0.1:80"},
		MaxRetryCount: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	c.SetOutput(&logs)
	c.Info = true
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := c.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := c.Wait(); err == nil {
		t.Fatal("client with a rejected forward remote kept running")
	}
	if !strings.Contains(logs.String(), forwardDisabledMsg) {
		t.Fatalf("client log does not show the rejection:\n%s", logs.String())
	}
}

// TestOutboundChannelsRejectedWithoutAllowForward opens channels directly,
// skipping the config validation, to confirm the tunnel refuses them.
func TestOutboundChannelsRejectedWithoutAllowForward(t *testing.T) {
	target, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	accepted := make(chan struct{}, 1)
	go func() {
		for {
			conn, err := target.Accept()
			if err != nil {
				return
			}
			conn.Close()
			select {
			case accepted <- struct{}{}:
			default:
			}
		}
	}()

	addr := startForwardTestServer(t, &chserver.Config{
		KeySeed: "allow-forward-channel",
		Reverse: true,
		Socks5:  true,
	})
	sc, _, _ := dialChiselSSH(t, addr, "", "")
	defer sc.Close()
	//config without forward remotes is valid
	sendConfig(t, sc, []*settings.Remote{})

	for _, dest := range []string{target.Addr().String(), target.Addr().String() + "/udp", "socks"} {
		ch, _, err := sc.OpenChannel("chisel", []byte(dest))
		if err == nil {
			ch.Close()
			t.Fatalf("channel to %q was accepted", dest)
		}
		if !strings.Contains(err.Error(), "Denied outbound connection") {
			t.Fatalf("channel to %q rejected for the wrong reason: %v", dest, err)
		}
	}
	select {
	case <-accepted:
		t.Fatal("server dialed the target")
	case <-time.After(200 * time.Millisecond):
	}
}
