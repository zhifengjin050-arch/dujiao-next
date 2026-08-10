package app

import (
	"context"
	"errors"
	"os/signal"
	"time"

	"go.uber.org/zap"
)

// Service ????
type Service interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// Runner ?????
type Runner struct {
	services []Service
}

// NewRunner ???????
func NewRunner(services ...Service) *Runner {
	return &Runner{services: services}
}

// RunWithOptions ???????????
func RunWithOptions(runner *Runner, opts Options) error {
	if runner == nil {
		return errors.New("runner is nil")
	}
	opts = normalizeOptions(opts)
	ctx := context.Background()
	if len(opts.Signals) > 0 {
		var cancel context.CancelFunc
		ctx, cancel = signal.NotifyContext(ctx, opts.Signals...)
		defer cancel()
	}

	err := runner.Run(ctx, opts.ShutdownTimeout, opts.Logger)
	return err
}

// Run ???????
func (r *Runner) Run(ctx context.Context, stopTimeout time.Duration, logger *zap.SugaredLogger) error {
	if r == nil || len(r.services) == 0 {
		return errors.New("no services to run")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, len(r.services))
	for _, svc := range r.services {
		service := svc
		go func() {
			name := "unknown"
			if service != nil {
				name = service.Name()
			}
			if logger != nil {
				logger.Infow("service_start", "service", name)
			}
			if service == nil {
				errCh <- errors.New("service is nil")
				return
			}
			errCh <- service.Start(ctx)
			if logger != nil {
				logger.Infow("service_exit", "service", name)
			}
		}()
	}

	var runErr error
	select {
	case <-ctx.Done():
		runErr = ctx.Err()
	case err := <-errCh:
		runErr = err
	}

	cancel()
	if stopTimeout <= 0 {
		stopTimeout = 10 * time.Second
	}
	stopCtx, stopCancel := context.WithTimeout(context.Background(), stopTimeout)
	defer stopCancel()
	for _, svc := range r.services {
		if svc == nil {
			continue
		}
		if err := svc.Stop(stopCtx); err != nil {
			if logger != nil {
				logger.Errorw("service_stop_failed", "service", svc.Name(), "error", err)
			}
		}
	}
	if errors.Is(runErr, context.Canceled) {
		return nil
	}
	return runErr
}
