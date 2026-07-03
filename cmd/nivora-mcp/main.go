package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	appmcp "github.com/sevoniva/nivora/internal/app/mcp"
)

func main() {
	configPath := flag.String("config", "configs/server.yaml", "config file path")
	stdio := flag.Bool("stdio", true, "serve MCP over stdio")
	flag.Parse()

	if !*stdio {
		fmt.Fprintln(os.Stderr, "only --stdio mode is supported in this MCP foundation")
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	if err := appmcp.RunStdio(ctx, *configPath, os.Stdin, os.Stdout, logger); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
