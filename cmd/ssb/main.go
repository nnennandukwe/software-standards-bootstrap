package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/cli"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return cli.RunContext(ctx, os.Args[1:], os.Stdout, os.Stderr)
}
