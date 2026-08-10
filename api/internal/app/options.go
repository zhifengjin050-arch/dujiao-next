package app

import (
	"os"
	"time"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/logger"

	"go.uber.org/zap"
)

const (
	ModeAll    = "all"
	ModeAPI    = "api"
	ModeWorker = "worker"
)

// Options ??????
type Options struct {
	Config          *config.Config
	Logger          *zap.SugaredLogger
	Signals         []os.Signal
	ShutdownTimeout time.Duration
	Mode            string
}

// normalizeOptions ??????
func normalizeOptions(opts Options) Options {
	if opts.Logger == nil {
		opts.Logger = logger.S()
	}
	if opts.ShutdownTimeout <= 0 {
		opts.ShutdownTimeout = 10 * time.Second
	}
	if opts.Mode == "" {
		opts.Mode = ModeAll
	}
	return opts
}
