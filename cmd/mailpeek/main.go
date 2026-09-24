package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/mailpeek/mailpeek/internal/app"
	"github.com/mailpeek/mailpeek/internal/config"
	"github.com/mailpeek/mailpeek/web"
)

var version = "dev"

func main() {
	started := time.Now()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Args[1:], os.Getenv, os.Stdout, started)
	stop()
	if err != nil && !errors.Is(err, config.ErrHelp) {
		fmt.Fprintln(os.Stderr, "mailpeek:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, getenv func(string) string, out io.Writer, started time.Time) error {
	if len(args) > 0 {
		switch args[0] {
		case "version", "--version", "-v":
			fmt.Fprintln(out, "mailpeek", version)
			return nil
		case "healthcheck":
			return healthcheck(args[1:], getenv)
		case "send":
			return send(args[1:], getenv, out)
		}
	}
	cfg, err := config.Load(args, getenv, out)
	if err != nil {
		return err
	}

	logger := log.New(os.Stderr, "mailpeek: ", log.LstdFlags)
	ui := web.FS()
	a := app.New(cfg, ui, version, logger)
	if err := a.Start(); err != nil {
		return err
	}
	printBanner(out, cfg, a, ui != nil, time.Since(started))

	select {
	case <-ctx.Done():
	case err = <-a.Err():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return errors.Join(err, a.Shutdown(shutdownCtx))
}

func printBanner(out io.Writer, cfg config.Config, a *app.App, hasUI bool, elapsed time.Duration) {
	display := cfg.Host
	if display == "" || display == "0.0.0.0" || display == "::" {
		display = "localhost"
	}
	login := func(c config.Credentials) string {
		if c.Enabled() {
			return "  (login required)"
		}
		return ""
	}
	web := login(cfg.UIAuth)
	if !hasUI {
		web += "  (API only: this binary was built without the Web UI, e.g. by go install;" +
			" use a release binary, the Docker image or make build)"
	}
	fmt.Fprintf(out, "Mailpeek\n\nSMTP  smtp://%s%s\nWeb   http://%s%s\n\nReady in %dms\n",
		net.JoinHostPort(display, port(a.SMTPAddr())), login(cfg.SMTPAuth),
		net.JoinHostPort(display, port(a.HTTPAddr())), web,
		elapsed.Milliseconds())
}

func port(addr net.Addr) string {
	if tcp, ok := addr.(*net.TCPAddr); ok {
		return strconv.Itoa(tcp.Port)
	}
	return addr.String()
}

func healthcheck(args []string, getenv func(string) string) error {
	cfg, err := config.Load(args, getenv, io.Discard)
	if err != nil {
		return err
	}
	client := http.Client{Timeout: 2 * time.Second}
	url := fmt.Sprintf("http://127.0.0.1:%d/api/v1/health", cfg.HTTPPort)
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("no Mailpeek answering on %s: is it running, and on this HTTP port "+
			"(--http-port or MAILPEEK_HTTP_PORT)? %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unhealthy: HTTP %d", resp.StatusCode)
	}
	return nil
}
