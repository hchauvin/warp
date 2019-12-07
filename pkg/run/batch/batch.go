package batch

import (
	"context"
	"fmt"
	"github.com/hchauvin/name_manager/pkg/name_manager"
	"github.com/hchauvin/warp/pkg/batches"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/deploy"
	"github.com/hchauvin/warp/pkg/k8s"
	"github.com/hchauvin/warp/pkg/pipelines"
	"github.com/hchauvin/warp/pkg/proc"
	"github.com/hchauvin/warp/pkg/run"
	"github.com/hchauvin/warp/pkg/run/env"
	"github.com/hchauvin/warp/pkg/stacks"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"go.uber.org/atomic"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"os"
	"strings"
	"sync"
)

const logDomain = "batch"

type RunBatchOptions struct {
	Parallelism int
	MaxStacksPerPipeline int
	Bail bool
}

type completionStatus int

const (
	pending completionStatus = iota
	success
	skipped
)

func RunBatch(
	ctx context.Context,
	cfg *config.Config,
	batch *batches.Batch,
	options *RunBatchOptions,
	k8sClient *k8s.K8s,
) error {
	completed := make(map[string]chan struct{})
	completionStatus := make(map[string]completionStatus)
	var completionMut sync.RWMutex
	for _, cmd := range batch.Commands {
		completionStatus[cmd.Name] = pending
		completed[cmd.Name] = make(chan struct{})
	}

	runner := &runner{
		cfg: cfg,
		k8sClient: k8sClient,
		options: options,
		pipelines: make(map[string]*pipeline),
	}
	defer runner.clean()

	{
		var g errgroup.Group
		var mut sync.Mutex
		for _, batchPipeline := range batch.Pipelines {
			batchPipeline := batchPipeline
			g.Go(func() error {
				p, err := pipelines.Read(cfg, batchPipeline.Path)
				if err != nil {
					return err
				}
				mut.Lock()
				defer mut.Unlock()
				runner.pipelines[batchPipeline.Name] = &pipeline{
					batchPipeline: batchPipeline,
					pipeline:      p,
					releasec:      make(chan struct{}),
					stacks: make(map[string]*stackInfo),
				}
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return err
		}
	}

	cfg.Logger().Info(
		logDomain,
		"%d pipelines, %d commands -- parallelism: %d",
		len(runner.pipelines),
		len(batch.Commands),
		options.Parallelism)

	cmdSema := semaphore.NewWeighted(int64(options.Parallelism))

	g, gctx := errgroup.WithContext(ctx)
	for _, cmd := range batch.Commands {
		cmd := cmd
		g.Go(func() error {
			for _, dep := range cmd.DependsOn {
				completionMut.RLock()
				status, ok := completionStatus[dep]
				completionMut.RUnlock()
				if !ok || status == skipped {
					completionMut.Lock()
					completionStatus[cmd.Name] = skipped
					completionMut.Unlock()
					close(completed[cmd.Name])
					return nil
				}
				select {
				case <-gctx.Done():
					return gctx.Err()
				case <-completed[dep]:
					completionMut.RLock()
					status := completionStatus[dep]
					completionMut.RUnlock()
					if status == skipped {
						completionMut.Lock()
						completionStatus[cmd.Name] = skipped
						completionMut.Unlock()
						close(completed[cmd.Name])
						return nil
					}
				}
			}

			if err := cmdSema.Acquire(gctx, 1); err != nil {
				return err
			}
			defer cmdSema.Release(1)

			runner.cfg.Logger().Info(logDomain, "command %s: start", cmd.Name)

			if err := runner.execCommand(gctx, cfg, &cmd, k8sClient); err != nil {
				return fmt.Errorf("command %s: %s", cmd.Name, err)
			}

			completionMut.Lock()
			completionStatus[cmd.Name] = success
			completionMut.Unlock()
			close(completed[cmd.Name])

			runner.cfg.Logger().Info(logDomain, "command %s: success", cmd.Name)

			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	if len(runner.errored) > 0 {
		return fmt.Errorf("The following commands errored: %s", strings.Join(runner.errored, " "))
	}
	return nil
}

type runner struct {
	cfg *config.Config
	k8sClient *k8s.K8s
	options *RunBatchOptions
	pipelines map[string]*pipeline
	stacksMut sync.Mutex
	errored []string
	erroredMut sync.Mutex
}

type pipeline struct {
	batchPipeline batches.Pipeline
	pipeline *pipelines.Pipeline
	stackCount atomic.Int64
	freeStackCount atomic.Int64
	releasec chan struct{}
	stacks map[string]*stackInfo
}

type stackInfo struct {
	pipelineName string
	name names.Name
	release name_manager.ReleaseFunc
	trans *env.Transformer
	exclusiveLock bool
	usageCount int
	deployed atomic.Bool
	before atomic.Bool
	initialized chan struct{}
}

func (runner *runner) clean() {
	// Release the stacks
	var g sync.WaitGroup
	for _, pipeline := range runner.pipelines {
		for _, stack := range pipeline.stacks {
			g.Add(1)
			go func() {
				defer g.Done()
				stack.release()
			}()
		}
	}
	g.Wait()
}

func (runner *runner) hold(ctx context.Context, pipelineName string, exclusive bool) (*stackInfo, error) {
	pipeline, err := runner.pipeline(pipelineName)
	if err != nil {
		return nil, err
	}

	var wait bool
	if exclusive {
		wait = pipeline.stackCount.Load() >= int64(runner.options.MaxStacksPerPipeline)
	} else {
		wait = pipeline.freeStackCount.Load() == 0 &&
			pipeline.stackCount.Load() >= int64(runner.options.MaxStacksPerPipeline)
	}
	if wait {
		runner.cfg.Logger().Info(
			logDomain,
			"max stacks per pipeline %d reached; waiting for a stack release",
			runner.options.MaxStacksPerPipeline)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-pipeline.releasec:
		}
	}

	runner.stacksMut.Lock()
	defer runner.stacksMut.Unlock()

	var stack *stackInfo
	if exclusive {
		for _, st := range pipeline.stacks {
			if st.usageCount == 0 {
				stack = st
				break
			}
		}
		if stack != nil {
			stack.usageCount = 1
			stack.exclusiveLock = true
			pipeline.freeStackCount.Dec()
			return stack, nil
		}
	} else {
		for _, st := range pipeline.stacks {
			if !st.exclusiveLock {
				stack = st
				break
			}
		}
		if stack != nil {
			stack.usageCount++
			return stack, nil
		}
	}

	name, release, err := stacks.Hold(runner.cfg, pipeline.pipeline)
	if err != nil {
		return nil, err
	}

	stack = &stackInfo{
		pipelineName: pipelineName,
		name: *name,
		release: release,
		trans: env.NewTranformer(runner.cfg, *name, runner.k8sClient),
		usageCount: 1,
		exclusiveLock: exclusive,
		initialized: make(chan struct{}),
	}
	pipeline.stacks[name.String()] = stack
	pipeline.stackCount.Inc()
	if !exclusive {
		pipeline.freeStackCount.Inc()
	}
	return stack, nil
}

func (runner *runner) release(pipelineName, stackName string) {
	runner.stacksMut.Lock()
	defer runner.stacksMut.Unlock()

	stack := runner.pipelines[pipelineName].stacks[stackName]
	stack.usageCount--
	if stack.exclusiveLock {
		stack.before.Store(false)
		runner.pipelines[stack.pipelineName].freeStackCount.Inc()
	}
	stack.exclusiveLock = false
	go func() {
		<- runner.pipelines[stack.pipelineName].releasec
	}()
}

func (runner *runner) execCommand(
	ctx context.Context,
	cfg *config.Config,
	cmd *batches.BatchCommand,
	k8sClient *k8s.K8s,
) error {
	// Hold the stacks
	g, gctx := errgroup.WithContext(ctx)
	var stacks []*stackInfo
	var stacksMut sync.Mutex
	fmt.Printf("YOOO! %v\n", cmd.Pipelines)
	for _, pipelineName := range cmd.Pipelines {
		pipelineName := pipelineName
		g.Go(func() error {
			stack, err := runner.hold(gctx, pipelineName, cmd.Exclusive)
			if err != nil {
				return err
			}

			if !stack.deployed.Swap(true) {
				pipeline, err := runner.pipeline(stack.pipelineName)
				if err != nil {
					return err
				}
				if err := deploy.Exec(gctx, cfg, pipeline.pipeline, stack.name, k8sClient); err != nil {
					return fmt.Errorf("deploy failed for stack %s: %v", stack.name, err)
				}
			}

			if !stack.before.Swap(true) {
				pipeline, err := runner.pipeline(stack.pipelineName)
				if err != nil {
					return err
				}
				if pipeline.batchPipeline.Env != "" {
					e, err := pipeline.pipeline.Environments.Get(pipeline.batchPipeline.Env)
					if err != nil {
						return err
					}
					err = run.ExecHooks(
						ctx,
						cfg,
						stack.name,
						"before",
						e.Before,
						runner.k8sClient)
					if err != nil {
						return err
					}
				}
				close(stack.initialized)
			}

			select {
			case <-gctx.Done():
				return gctx.Err()
			case <-stack.initialized:
			}

			stacksMut.Lock()
			stacks = append(stacks, stack)
			stacksMut.Unlock()
			return nil
		})
	}
	defer func() {
		for _, stack := range stacks {
			runner.release(stack.pipelineName, stack.name.String())
		}
	}()
	if err := g.Wait(); err != nil {
		return err
	}

	// Gather environment variables
	var allEnv []string
	var allEnvMut sync.Mutex
	g, gctx = errgroup.WithContext(ctx)
	for _, stack := range stacks {
		stack := stack
		g.Go(func() error {
			pipeline, err := runner.pipeline(stack.pipelineName)
			if err != nil {
				return err
			}

			genv, genvctx := errgroup.WithContext(gctx)
			if pipeline.batchPipeline.Env != "" {
				env, err := pipeline.pipeline.Environments.Get(pipeline.batchPipeline.Env)
				if err != nil {
					return err
				}
				for _, e := range env.Env {
					e := e
					genv.Go(func() error {
						s, err := stack.trans.Get(genvctx, e)
						if err != nil {
							return err
						}
						allEnvMut.Lock()
						defer allEnvMut.Unlock()
						allEnv = append(allEnv, s)
						return nil
					})
				}
			}
			return genv.Wait()
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	allEnv = append(allEnv, cmd.Env...)

	var tries int
	if cmd.Flaky {
		tries = 3
	} else {
		tries = 1
	}

	var err error
	for {
		procCmd := proc.GracefulCommandContext(ctx, cmd.Command[0], cmd.Command[1:]...)
		if cmd.WorkingDir != "" {
			procCmd.Dir = cfg.Path(cmd.WorkingDir)
		}

		cfg.Logger().Info("run:"+cmd.Name+":env", "%s", strings.Join(allEnv, "\n"))
		procCmd.Env = append(os.Environ(), allEnv...)
		cfg.Logger().Pipe("run:"+cmd.Name, procCmd)
		err = procCmd.Run()
		if err == nil {
			break
		}
		tries--
		if tries == 0 {
			break
		}
	}

	if err != nil {
		if runner.options.Bail {
			return fmt.Errorf("could not run '%s': %v", cmd.Name, err)
		}
		runner.erroredMut.Lock()
		runner.errored = append(runner.errored, cmd.Name)
		runner.erroredMut.Unlock()
	}
	return nil
}

func (runner *runner) pipeline(name string) (*pipeline, error) {
	pipeline, ok := runner.pipelines[name]
	if !ok {
		return nil, fmt.Errorf("cannot find pipeline with name '%s'", name)
	}
	return pipeline, nil
}
