package main

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/gosuri/eve/cmd"
	"github.com/gosuri/eve/logger"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, cancel := newContext()
	defer cancel()
	if err := cmd.NewRootCMD(ctx, cancel).Execute(); err != nil {
		logger.Debugf("%+v", err)
		return 1
	}
	return 0
}

func newContext() (context.Context, context.CancelFunc) {
	signals := []os.Signal{os.Interrupt}
	if runtime.GOOS != "windows" {
		signals = append(signals, syscall.SIGTERM)
	}
	return signal.NotifyContext(context.Background(), signals...)
}
