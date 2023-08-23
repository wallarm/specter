package engine

import (
	"context"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"github.com/wallarm/specter/core"
	"github.com/wallarm/specter/core/coreutil"
	"github.com/wallarm/specter/core/warmup"
	"github.com/wallarm/specter/lib/errutil"
	"github.com/wallarm/specter/lib/monitoring"
	"go.uber.org/zap"
)

type Config struct {
	Pools []InstancePoolConfig `config:"pools" validate:"required,dive"`
}

type InstancePoolConfig struct {
	ID              string
	Provider        core.Provider                 `config:"ammo" validate:"required"`
	Aggregator      core.Aggregator               `config:"result" validate:"required"`
	NewGun          func() (core.Gun, error)      `config:"gun" validate:"required"`
	RPSPerInstance  bool                          `config:"rps-per-instance"`
	NewRPSSchedule  func() (core.Schedule, error) `config:"rps" validate:"required"`
	StartupSchedule core.Schedule                 `config:"startup" validate:"required"`
	DiscardOverflow bool                          `config:"discard_overflow"`
}

// TODO: use something github.com/rcrowley/go-metrics based.
// Its high level primitives like Meter can be not fast enough, but EWMAs
// and Counters should good for that.
type Metrics struct {
	Request        *monitoring.Counter
	Response       *monitoring.Counter
	InstanceStart  *monitoring.Counter
	InstanceFinish *monitoring.Counter
}

func New(log *zap.Logger, m Metrics, conf Config) *Engine {
	return &Engine{log: log, config: conf, metrics: m}
}

type Engine struct {
	log     *zap.Logger
	config  Config
	metrics Metrics
	wait    sync.WaitGroup
}

// Run runs all instance pools. Run blocks until fail happen, or all pools
// subroutines are successfully finished.
// Ctx will be ancestor to Contexts passed to AmmoQueue, Gun and Aggregator.
// That's ctx cancel cancels shooting and it's Context values can be used for communication between plugins.
func (e *Engine) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		e.log.Info("Engine finished")
		cancel()
	}()

	runRes := make(chan poolRunResult, 1)
	for i, conf := range e.config.Pools {
		if conf.ID == "" {
			conf.ID = fmt.Sprintf("pool_%v", i)
		}
		e.wait.Add(1)
		pool := newPool(e.log, e.metrics, e.wait.Done, conf)
		go func() {
			err := pool.Run(ctx)
			select {
			case runRes <- poolRunResult{pool.ID, err}:
			case <-ctx.Done():
				pool.log.Info("Pool run result suppressed",
					zap.String("id", pool.ID), zap.Error(err))
			}
		}()
	}

	for i := 0; i < len(e.config.Pools); i++ {
		select {
		case res := <-runRes:
			e.log.Debug("Pool awaited", zap.Int("awaited", i),
				zap.String("id", res.ID), zap.Error(res.Err))
			if res.Err != nil {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}
				return errors.WithMessage(res.Err, fmt.Sprintf("%q pool run failed", res.ID))
			}
		case <-ctx.Done():
			e.log.Info("Engine run canceled")
			return ctx.Err()
		}
	}
	return nil
}

// Wait blocks until all run engine tasks are finished.
// Useful only in case of fail, because successful run awaits all started tasks.
func (e *Engine) Wait() {
	e.wait.Wait()
}

func newPool(log *zap.Logger, m Metrics, onWaitDone func(), conf InstancePoolConfig) *instancePool {
	log = log.With(zap.String("pool", conf.ID))
	return &instancePool{log, m, onWaitDone, conf, nil}
}

type instancePool struct {
	log        *zap.Logger
	metrics    Metrics
	onWaitDone func()
	InstancePoolConfig
	gunWarmUpResult interface{}
}

// Run start instance pool. Run blocks until fail happen, or all instances finish.
// What's going on:
// AmmoQueue and Aggregator are started in separate goroutines.
// Instances create due to schedule is started in separate goroutine.
// Every new instance started in separate goroutine.
// When all instances are finished, Aggregator and AmmoQueue contexts are canceled,
// and their execution results are awaited.
// If error happen or Run context has been canceled, Run returns non-nil error immediately,
// remaining results awaiting goroutine in background, that will call onWaitDone callback,
// when all started subroutines will be finished.
func (p *instancePool) Run(ctx context.Context) error {
	p.log.Info("Pool run started")
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		p.log.Info("Pool run finished")
		cancel()
	}()

	if err := p.warmUpGun(ctx); err != nil {
		p.onWaitDone()
		return err
	}

	rh, err := p.runAsync(ctx)
	if err != nil {
		return err
	}

	awaitErr := p.awaitRunAsync(rh)

	select {
	case <-ctx.Done():
		p.log.Info("Pool execution canceled")
		return ctx.Err()
	case err, ok := <-awaitErr:
		if ok {
			p.log.Info("Pool failed. Canceling started tasks", zap.Error(err))
			return err
		}
		p.log.Info("Pool run finished successfully")
		return nil
	}
}

func (p *instancePool) warmUpGun(ctx context.Context) error {
	gun, err := p.NewGun()
	if err != nil {
		return fmt.Errorf("can't initiate a gun: %w", err)
	}
	if gunWithWarmUp, ok := gun.(warmup.WarmedUp); ok {
		p.gunWarmUpResult, err = gunWithWarmUp.WarmUp(&warmup.Options{
			Log: p.log,
			Ctx: ctx,
		})
		if err != nil {
			return fmt.Errorf("gun warm up failed: %w", err)
		}
	}
	return nil
}

type poolAsyncRunHandle struct {
	runCtx              context.Context
	runCancel           context.CancelFunc
	instanceStartCtx    context.Context
	instanceStartCancel context.CancelFunc

	providerErr   <-chan error
	aggregatorErr <-chan error
	startRes      <-chan startResult
	// Read only actually. But can be closed by reader, to be sure, that no result has been lost.
	runRes chan instanceRunResult
}

func (p *instancePool) runAsync(runCtx context.Context) (*poolAsyncRunHandle, error) {
	// Canceled in case all instances finish, fail or run runCancel.
	runCtx, runCancel := context.WithCancel(runCtx)
	_ = runCancel
	// Canceled also on out of ammo, and finish of shared RPS schedule.
	instanceStartCtx, instanceStartCancel := context.WithCancel(runCtx)
	newInstanceSchedule, err := p.buildNewInstanceSchedule(instanceStartCtx, instanceStartCancel)
	if err != nil {
		return nil, err
	}
	// Seems good enough. Even if some run will block on result send, it's not real problem.
	const runResultBufSize = 64
	var (
		// All channels are buffered. All results should be read.
		providerErr   = make(chan error, 1)
		aggregatorErr = make(chan error, 1)
		startRes      = make(chan startResult, 1)
		runRes        = make(chan instanceRunResult, runResultBufSize)
	)
	go func() {
		deps := core.ProviderDeps{Log: p.log, PoolID: p.ID}
		providerErr <- p.Provider.Run(runCtx, deps)
	}()
	go func() {
		deps := core.AggregatorDeps{Log: p.log}
		aggregatorErr <- p.Aggregator.Run(runCtx, deps)
	}()
	go func() {
		started, err := p.startInstances(instanceStartCtx, runCtx, newInstanceSchedule, runRes)
		startRes <- startResult{started, err}
	}()
	return &poolAsyncRunHandle{
		runCtx:              runCtx,
		runCancel:           runCancel,
		instanceStartCtx:    instanceStartCtx,
		instanceStartCancel: instanceStartCancel,
		providerErr:         providerErr,
		aggregatorErr:       aggregatorErr,
		runRes:              runRes,
		startRes:            startRes,
	}, nil
}

func (p *instancePool) awaitRunAsync(runHandle *poolAsyncRunHandle) <-chan error {
	ah, awaitErr := p.newAwaitRunHandle(runHandle)
	go func() {
		defer func() {
			ah.log.Debug("Pool wait finished")
			close(ah.awaitErr)
			if p.onWaitDone != nil {
				p.onWaitDone()
			}
		}()
		ah.awaitRun()
	}()
	return awaitErr
}

type runAwaitHandle struct {
	log *zap.Logger
	poolAsyncRunHandle
	awaitErr         chan<- error
	toWait           int
	startedInstances int
	awaitedInstances int
}

func (p *instancePool) newAwaitRunHandle(runHandle *poolAsyncRunHandle) (*runAwaitHandle, <-chan error) {
	awaitErr := make(chan error)
	const resultsToWait = 4 // AmmoQueue, Aggregator, instance start, instance run.
	awaitHandle := &runAwaitHandle{
		log:                p.log,
		poolAsyncRunHandle: *runHandle,
		awaitErr:           awaitErr,
		toWait:             resultsToWait,
		startedInstances:   -1, // Undefined until start finish.
	}
	return awaitHandle, awaitErr
}

func (ah *runAwaitHandle) awaitRun() {
	for ah.toWait > 0 {
		select {
		case err := <-ah.providerErr:
			ah.providerErr = nil
			// TODO: not wait for provider, to return success result?
			ah.toWait--
			ah.log.Debug("AmmoQueue awaited", zap.Error(err))
			if !errutil.IsCtxError(ah.runCtx, err) {
				ah.onErrAwaited(errors.WithMessage(err, "provider failed"))
			}
		case err := <-ah.aggregatorErr:
			ah.aggregatorErr = nil
			ah.toWait--
			ah.log.Debug("Aggregator awaited", zap.Error(err))
			if !errutil.IsCtxError(ah.runCtx, err) {
				ah.onErrAwaited(errors.WithMessage(err, "aggregator failed"))
			}
		case res := <-ah.startRes:
			ah.startRes = nil
			ah.toWait--
			ah.startedInstances = res.Started
			ah.log.Debug("Instances start awaited", zap.Int("started", ah.startedInstances), zap.Error(res.Err))
			if !errutil.IsCtxError(ah.instanceStartCtx, res.Err) {
				ah.onErrAwaited(errors.WithMessage(res.Err, "instances start failed"))
			}
			ah.checkAllInstancesAreFinished() // There is a race between run and start results.
		case res := <-ah.runRes:
			ah.awaitedInstances++
			if ent := ah.log.Check(zap.DebugLevel, "Instance run awaited"); ent != nil {
				ent.Write(zap.Int("id", res.ID), zap.Int("awaited", ah.awaitedInstances), zap.Error(res.Err))
			}

			if res.Err == outOfAmmoErr {
				if !ah.isStartFinished() {
					ah.log.Debug("Canceling instance start because out of ammo")
					ah.instanceStartCancel()
				}
			} else if !errutil.IsCtxError(ah.runCtx, res.Err) {
				ah.onErrAwaited(errors.WithMessage(res.Err, fmt.Sprintf("instance %q run failed", res.ID)))
			}
			ah.checkAllInstancesAreFinished()
		}
	}
}

func (ah *runAwaitHandle) onErrAwaited(err error) {
	select {
	case ah.awaitErr <- err:
	case <-ah.runCtx.Done():
		if err != ah.runCtx.Err() {
			ah.log.Debug("Error suppressed after run cancel", zap.Error(err))
		}
	}
}

func (ah *runAwaitHandle) checkAllInstancesAreFinished() {
	allFinished := ah.isStartFinished() && ah.awaitedInstances >= ah.startedInstances
	if !allFinished {
		return
	}
	// Assert, that all run results are awaited.
	close(ah.runRes)
	res, ok := <-ah.runRes
	if ok {
		ah.log.Panic("Unexpected run result", zap.Any("res", res))
	}

	ah.runRes = nil
	ah.toWait--
	ah.log.Info("All instances runs awaited.", zap.Int("awaited", ah.awaitedInstances))
	ah.runCancel() // Signal to provider and aggregator, that pool run is finished.

}

func (ah *runAwaitHandle) isStartFinished() bool {
	return ah.startRes == nil
}

func (p *instancePool) startInstances(
	startCtx, runCtx context.Context,
	newInstanceSchedule func() (core.Schedule, error),
	runRes chan<- instanceRunResult) (started int, err error) {
	deps := instanceDeps{
		newInstanceSchedule,
		p.NewGun,
		instanceSharedDeps{p.Provider, p.metrics, p.gunWarmUpResult, p.Aggregator, p.DiscardOverflow},
	}

	waiter := coreutil.NewWaiter(p.StartupSchedule, startCtx)

	// If create all instances asynchronously, and creation will fail, too many errors appears in log.
	ok := waiter.Wait()
	if !ok {
		err = startCtx.Err()
		return
	}
	firstInstance, err := newInstance(runCtx, p.log, p.ID, 0, deps)
	if err != nil {
		return
	}
	started++
	go func() {
		runRes <- instanceRunResult{0, func() error {
			defer firstInstance.Close()
			return firstInstance.Run(runCtx)
		}()}
	}()

	for ; waiter.Wait(); started++ {
		id := started
		go func() {
			runRes <- instanceRunResult{id, runNewInstance(runCtx, p.log, p.ID, id, deps)}
		}()
	}
	err = startCtx.Err()
	return
}

func (p *instancePool) buildNewInstanceSchedule(startCtx context.Context, cancelStart context.CancelFunc) (
	func() (core.Schedule, error), error,
) {
	if p.RPSPerInstance {
		return p.NewRPSSchedule, nil
	}
	sharedRPSSchedule, err := p.NewRPSSchedule()
	if err != nil {
		return nil, err
	}
	sharedRPSSchedule = coreutil.NewCallbackOnFinishSchedule(sharedRPSSchedule, func() {
		select {
		case <-startCtx.Done():
			p.log.Debug("RPS schedule has been finished")
			return
		default:
			p.log.Info("RPS schedule has been finished. Canceling instance start.")
			cancelStart()
		}
	})
	return func() (core.Schedule, error) {
		return sharedRPSSchedule, err
	}, nil
}

func runNewInstance(ctx context.Context, log *zap.Logger, poolID string, id int, deps instanceDeps) error {
	instance, err := newInstance(ctx, log, poolID, id, deps)
	if err != nil {
		return err
	}
	defer instance.Close()
	return instance.Run(ctx)
}

type poolRunResult struct {
	ID  string
	Err error
}

type instanceRunResult struct {
	ID  int
	Err error
}

type startResult struct {
	Started int
	Err     error
}
