package main

import (
	"context"
	"os"
	"os/signal"

	"gh-mutual-follow/internal/cli"
	"gh-mutual-follow/internal/github"
	"golang.org/x/term"
)

var version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	code := cli.Run(ctx, os.Args[1:], github.NewClient(), cli.IO{
		In: os.Stdin, Out: os.Stdout, Err: os.Stderr,
		InputTTY: term.IsTerminal(int(os.Stdin.Fd())),
		ErrorTTY: term.IsTerminal(int(os.Stderr.Fd())),
	}, version)
	stop()
	os.Exit(code)
}
