package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gravitational/trace"

	"github.com/jsabo/teleport-msteams-webhook/internal/bot"
	"github.com/jsabo/teleport-msteams-webhook/internal/config"
	"github.com/jsabo/teleport-msteams-webhook/internal/plugin"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	configPath := flag.String("config", "/etc/teleport-msteams-webhook.toml", "path to TOML config file")
	dryRun := flag.Bool("dry-run", false, "validate config, connect to Teleport, POST a test card to each webhook URL, then exit")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("teleport-msteams-webhook %s (commit %s, built %s)\n", version, commit, date)
		return
	}

	if err := run(*configPath, *dryRun); err != nil {
		slog.Error("Fatal error", "error", err)
		os.Exit(1)
	}
}

func run(configPath string, dryRun bool) error {
	conf, err := config.LoadConfig(configPath)
	if err != nil {
		return trace.Wrap(err)
	}

	setupLogging(conf.Log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	clt, err := conf.NewClient(ctx)
	if err != nil {
		return trace.Wrap(err, "connecting to Teleport")
	}
	defer clt.Close()

	pong, err := clt.Ping(ctx)
	if err != nil {
		return trace.Wrap(err, "pinging Teleport cluster")
	}
	slog.InfoContext(ctx, "Connected to Teleport", "cluster", pong.ClusterName, "version", pong.ServerVersion)

	b := bot.New(pong.ProxyPublicAddr, conf.EffectiveLogoURL())

	if dryRun {
		return runDryRun(ctx, conf, b)
	}

	p := plugin.New(clt, b, conf.Recipients)
	slog.InfoContext(ctx, "Starting teleport-msteams-webhook", "version", version)
	return trace.Wrap(p.Run(ctx))
}

func runDryRun(ctx context.Context, conf *config.Config, b *bot.Bot) error {
	slog.InfoContext(ctx, "Dry-run: posting test card to all configured webhook URLs")

	seen := make(map[string]bool)
	for _, urls := range conf.Recipients {
		for _, u := range urls {
			seen[u] = true
		}
	}
	if len(seen) == 0 {
		slog.WarnContext(ctx, "No webhook URLs configured in role_to_recipients")
		return nil
	}

	testData := bot.RequestData{
		User:          "dry-run-test",
		Roles:         []string{"example-role"},
		RequestReason: "Connectivity test from teleport-msteams-webhook -dry-run",
	}

	var errs []error
	for webhookURL := range seen {
		if err := b.Post(ctx, webhookURL, "dry-run-000", testData); err != nil {
			slog.ErrorContext(ctx, "Failed", "url", webhookURL, "error", err)
			errs = append(errs, err)
		} else {
			slog.InfoContext(ctx, "OK", "url", webhookURL)
		}
	}

	if len(errs) > 0 {
		return trace.NewAggregate(errs...)
	}
	slog.InfoContext(ctx, "Dry-run complete — all webhooks reachable")
	return nil
}

func setupLogging(cfg config.LogConfig) {
	level := slog.LevelInfo
	switch cfg.Severity {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
}
