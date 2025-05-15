package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	chclient "github.com/jpillora/chisel/client"
	chserver "github.com/jpillora/chisel/server"
	chshare "github.com/jpillora/chisel/share"
	"github.com/jpillora/chisel/share/ccrypto"
	"github.com/jpillora/chisel/share/cos"
	"github.com/jpillora/chisel/share/settings"
)

var help = ` 
  Usage: chisel [command] [--help]

  Version: ` + chshare.BuildVersion + ` (` + runtime.Version() + `)

  Commands:
    server - runs chisel in server mode
    client - runs chisel in client mode

  Read more:
    https://github.com/jpillora/chisel
`

func main() {
	version := flag.Bool("version", false, "")
	v := flag.Bool("v", false, "")
	flag.Bool("help", false, "")
	flag.Bool("h", false, "")
	flag.Usage = func() {}
	flag.Parse()

	if *version || *v {
		fmt.Println(chshare.BuildVersion)
		os.Exit(0)
	}

	args := flag.Args()
	subcmd := ""
	if len(args) > 0 {
		subcmd = args[0]
		args = args[1:]
	}

	switch subcmd {
	case "server":
		server(args)
	case "client":
		client(args)
	default:
		fmt.Print(help)
		os.Exit(0)
	}
}

func generatePidFile() {
	pid := []byte(strconv.Itoa(os.Getpid()))
	if err := os.WriteFile("chisel.pid", pid, 0644); err != nil {
		log.Fatal(err)
	}
}

func server(args []string) {
	flags := flag.NewFlagSet("server", flag.ContinueOnError)
	config := &chserver.Config{}

	flags.StringVar(&config.KeySeed, "key", "", "")
	flags.StringVar(&config.KeyFile, "keyfile", "", "")
	flags.StringVar(&config.AuthFile, "authfile", "", "")
	flags.StringVar(&config.Auth, "auth", "", "")
	flags.DurationVar(&config.KeepAlive, "keepalive", 25*time.Second, "")
	flags.StringVar(&config.Proxy, "proxy", "", "")
	backend := flags.String("backend", "", "") // separate backend variable
	flags.BoolVar(&config.Socks5, "socks5", false, "")
	flags.BoolVar(&config.Reverse, "reverse", false, "")
	flags.StringVar(&config.TLS.Key, "tls-key", "", "")
	flags.StringVar(&config.TLS.Cert, "tls-cert", "", "")
	flags.Var(multiFlag{&config.TLS.Domains}, "tls-domain", "")
	flags.StringVar(&config.TLS.CA, "tls-ca", "", "")

	host := flags.String("host", "", "")
	p := flags.String("p", "", "")
	port := flags.String("port", "", "")
	pid := flags.Bool("pid", false, "")
	verbose := flags.Bool("v", false, "")
	keyGen := flags.String("keygen", "", "")

	flags.Usage = func() {
		fmt.Print("Use `chisel server [options]`")
		os.Exit(0)
	}
	if err := flags.Parse(args); err != nil {
		log.Fatal(err)
	}

	// Override Proxy with backend flag if provided
	if *backend != "" {
		config.Proxy = *backend
	}

	if *keyGen != "" {
		if err := ccrypto.GenerateKeyFile(*keyGen, config.KeySeed); err != nil {
			log.Fatal(err)
		}
		return
	}

	if config.KeySeed != "" {
		log.Print("Option `--key` is deprecated and will be removed in a future version of chisel.")
		log.Print("Please use `chisel server --keygen /file/path`, followed by `chisel server --keyfile /file/path` to specify the SSH private key")
	}

	if *host == "" {
		*host = os.Getenv("HOST")
	}
	if *host == "" {
		*host = "0.0.0.0"
	}
	if *port == "" {
		*port = *p
	}
	if *port == "" {
		*port = os.Getenv("PORT")
	}
	if *port == "" {
		*port = "8080"
	}
	if config.KeyFile == "" {
		config.KeyFile = settings.Env("KEY_FILE")
	} else if config.KeySeed == "" {
		config.KeySeed = settings.Env("KEY")
	}
	if config.Auth == "" {
		config.Auth = os.Getenv("AUTH")
	}

	s, err := chserver.NewServer(config)
	if err != nil {
		log.Fatal(err)
	}
	s.Debug = *verbose
	if *pid {
		generatePidFile()
	}
	go cos.GoStats()

	// Create a cancelable context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Handle interrupt signals to cancel the context gracefully
	go handleInterrupt(cancel)

	// Start the server in a goroutine
	go func() {
		if err := s.Start(*host, *port); err != nil {
			log.Fatal(err)
		}
	}()

	// Wait for cancellation signal
	<-ctx.Done()
	log.Println("Server shutting down")

	done := make(chan struct{})
	go func() {
		if err := s.Close(); err != nil {
			log.Printf("Error closing server: %v", err)
		}
		close(done)
	}()

	select {
	case <-done:
		log.Println("Server closed gracefully")
	case <-time.After(10 * time.Second):
		log.Println("Timeout waiting for server to close")
	}
}

func client(args []string) {
	flags := flag.NewFlagSet("client", flag.ContinueOnError)
	config := chclient.Config{Headers: http.Header{}}

	flags.StringVar(&config.Fingerprint, "fingerprint", "", "")
	flags.StringVar(&config.Auth, "auth", "", "")
	flags.DurationVar(&config.KeepAlive, "keepalive", 25*time.Second, "")
	flags.IntVar(&config.MaxRetryCount, "max-retry-count", -1, "")
	flags.DurationVar(&config.MaxRetryInterval, "max-retry-interval", 0, "")
	flags.StringVar(&config.Proxy, "proxy", "", "")
	flags.StringVar(&config.TLS.CA, "tls-ca", "", "")
	flags.BoolVar(&config.TLS.SkipVerify, "tls-skip-verify", false, "")
	flags.StringVar(&config.TLS.Cert, "tls-cert", "", "")
	flags.StringVar(&config.TLS.Key, "tls-key", "", "")
	flags.Var(&headerFlags{config.Headers}, "header", "")

	hostname := flags.String("hostname", "", "")
	sni := flags.String("sni", "", "")
	pid := flags.Bool("pid", false, "")
	verbose := flags.Bool("v", false, "")

	flags.Usage = func() {
		fmt.Print("Use `chisel client [options]`")
		os.Exit(0)
	}
	if err := flags.Parse(args); err != nil {
		log.Fatal(err)
	}

	args = flags.Args()
	if len(args) < 2 {
		log.Fatalf("A server and at least one remote is required")
	}
	config.Server = args[0]
	config.Remotes = args[1:]

	if config.Auth == "" {
		config.Auth = os.Getenv("AUTH")
	}
	if *hostname != "" {
		config.Headers.Set("Host", *hostname)
		config.TLS.ServerName = *hostname
	}
	if *sni != "" {
		config.TLS.ServerName = *sni
	}

	// Create a cancelable context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Handle interrupt signals to cancel the context gracefully
	go handleInterrupt(cancel)

	// Create and start the client
	c, err := chclient.NewClient(&config)
	if err != nil {
		log.Fatal(err)
	}
	c.Debug = *verbose
	if *pid {
		generatePidFile()
	}
	go cos.GoStats()

	// Start the client in a goroutine so we can control shutdown
	go func() {
		if err := c.Start(ctx); err != nil {
			log.Fatal(err)
		}
	}()

	// Wait for cancellation signal
	<-ctx.Done()
	log.Println("Client shutting down")

	done := make(chan struct{})
	go func() {
		if err := c.Close(); err != nil {
			log.Printf("Error closing client: %v", err)
		}
		close(done)
	}()

	select {
	case <-done:
		log.Println("Client closed gracefully")
	case <-time.After(10 * time.Second):
		log.Println("Timeout waiting for client to close")
	}
}

func handleInterrupt(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	fmt.Println("Received termination signal, initiating graceful shutdown...")
	cancel()
	fmt.Println("Shutting down gracefully...")
}

type multiFlag struct {
	values *[]string
}

func (flag multiFlag) String() string {
	return strings.Join(*flag.values, ", ")
}

func (flag multiFlag) Set(arg string) error {
	*flag.values = append(*flag.values, arg)
	return nil
}

type headerFlags struct {
	http.Header
}

func (flag *headerFlags) String() string {
	var out strings.Builder
	for k, v := range flag.Header {
		out.WriteString(fmt.Sprintf("%s: %s\n", k, v))
	}
	return out.String()
}

func (flag *headerFlags) Set(arg string) error {
	index := strings.Index(arg, ":")
	if index < 0 {
		return fmt.Errorf(`Invalid header (%s). Should be in the format "HeaderName: HeaderContent"`, arg)
	}
	if flag.Header == nil {
		flag.Header = http.Header{}
	}
	key := arg[0:index]
	value := strings.TrimSpace(arg[index+1:])
	flag.Header.Add(key, value)
	return nil
}
