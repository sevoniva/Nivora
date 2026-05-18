package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sevoniva/nivora/internal/app/runner"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	configPath := "configs/runner.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	if err := runner.Run(ctx, configPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
