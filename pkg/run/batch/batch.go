// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

// Package batch implements batch execution.
package batch

import (
	"bufio"
	"context"
	"fmt"
	"github.com/dustinkirkland/golang-petname"
	"github.com/hchauvin/name_manager/pkg/name_manager"
	"github.com/hchauvin/warp/pkg/batches"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/deploy"
	"github.com/hchauvin/warp/pkg/k8s"
	"github.com/hchauvin/warp/pkg/log/interactive"
	"github.com/hchauvin/warp/pkg/pipelines"
	"github.com/hchauvin/warp/pkg/proc"
	"github.com/hchauvin/warp/pkg/run"
	"github.com/hchauvin/warp/pkg/run/env"
	"github.com/hchauvin/warp/pkg/stacks"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

const logDomain = "batch"

// RunBatchOptions are the options for RunBatch.
type RunBatchOptions struct {
	Parallelism          int
	MaxStacksPerPipeline int
	Bail                 bool
	Advisory             bool
	Reporter             Reporter
	Events               chan<- interface{}
}

// Reporter is used by RunBatch to report on batch execution.
type Reporter interface {
	EnvironmentSetupResult(result *EnvironmentSetupResult)
	CommandOutput(info *CommandInfo) (io.WriteCloser, error)
	CommandResult(result *CommandResult)
	Finalize() error
}

// EnvironmentInfo gives info on an environment a batch command
// executed with.
type EnvironmentInfo struct {
	BatchID      string
	StackName    string
	PipelinePath string
}

// EnvironmentSetupResult contains the result of setting up
// an environment.
type EnvironmentSetupResult struct {
	EnvironmentInfo
	SetupType EnvironmentSetupType
	Err       *string
	Started   time.Time
	Completed time.Time
}

// EnvironmentSetupType is the type of environment setup for
// EnvironmentSetupResult.
type EnvironmentSetupType string

const (
	// EnvironmentDeployment is used when an environment is deployed.
	EnvironmentDeployment = EnvironmentSetupType("deployment")
	// EnvironmentInitialization is used when an environment is initialized.
	// Initialization occurs after deployment.
	EnvironmentInitialization = EnvironmentSetupType("initialization")
)

// CommandInfo describes a batch command for reporting purposes.
type CommandInfo struct {
	BatchID string
	Name    string
	Tries   int
}

// CommandResult gives the result of a command, for reporting purposes.
type CommandResult struct {
	CommandInfo
	Err       *string
	Started   time.Time
	Completed time.Time
}

type completionStatus int

const (
	pending completionStatus = iota
	success
	skipped
)

// RunBatch runs a batch of commands against stacks.
func RunBatch(
	ctx context.Context,
	cfg *config.Config,
	batch *batches.Batch,
	options *RunBatchOptions,
	k8sClient *k8s.K8s,
) error {
	defer func() {
		if err := options.Reporter.Finalize(); err != nil {
			cfg.Logger().Error(logDomain, "cannot finalize report: %v", err)
		}
	}()

	completed := make(map[string]chan struct{})
	completionStatus := make(map[string]completionStatus)
	var completionMut sync.RWMutex
	for _, cmd := range batch.Commands {
		completionStatus[cmd.Name] = pending
		completed[cmd.Name] = make(chan struct{})
	}

	batchID := petname.Generate(2, "-")
	runner := &runner{
		cfg:       cfg,
		k8sClient: k8sClient,
		options:   options,
		pipelines: make(map[string]*pipeline),
		trans:     make(map[string]*env.Transformer),
		sharedEnv: []string{
			"BATCH_ID=" + batchID,
		},
		batchID: batchID,
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
					stackHolder:   newStackHolder(),
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
		if runner.options.Advisory {
			runner.cfg.Logger().Warning(
				logDomain,
				"The following commands errored: %s",
				strings.Join(runner.errored, " "),
			)
		} else {
			return fmt.Errorf("The following commands errored: %s", strings.Join(runner.errored, " "))
		}
	}
	return nil
}

type runner struct {
	cfg        *config.Config
	k8sClient  *k8s.K8s
	options    *RunBatchOptions
	pipelines  map[string]*pipeline
	stacksMut  sync.Mutex
	trans      map[string]*env.Transformer
	transMut   sync.Mutex
	errored    []string
	erroredMut sync.Mutex
	sharedEnv  []string
	batchID    string
}

type pipeline struct {
	batchPipeline batches.Pipeline
	pipeline      *pipelines.Pipeline
	stackHolder   stackHolder
}

func (runner *runner) clean() {
	// Release the stacks
	var g sync.WaitGroup
	for _, pipeline := range runner.pipelines {
		for _, stack := range pipeline.stackHolder.stacks {
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

	return pipeline.stackHolder.hold(ctx, runner.cfg.Logger(), pipelineName, holdConfig{
		maxStacksPerPipeline: runner.options.MaxStacksPerPipeline,
		exclusive:            exclusive,
		hold: func() (*names.Name, <-chan error, name_manager.ReleaseFunc, error) {
			return stacks.Hold(runner.cfg, pipeline.pipeline)
		},
	})
}

func (runner *runner) release(pipelineName, stackName string) {
	runner.pipelines[pipelineName].stackHolder.release(stackName)
}

func (runner *runner) execCommand(
	ctx context.Context,
	cfg *config.Config,
	cmd *batches.BatchCommand,
	k8sClient *k8s.K8s,
) error {
	// Hold the stacks
	runner.event(interactive.SetStateEvent{
		Name:  cmd.Name,
		State: interactive.Started,
		Stage: "setup",
	})
	g, gctx := errgroup.WithContext(ctx)
	var stacks []*stackInfo
	var stacksMut sync.Mutex
	for _, pipelineName := range cmd.Pipelines {
		pipelineName := pipelineName
		g.Go(func() error {
			stack, err := runner.hold(gctx, pipelineName, cmd.Exclusive)
			if err != nil {
				return err
			}

			info := EnvironmentInfo{
				BatchID:      runner.batchID,
				StackName:    stack.name.DNSName(),
				PipelinePath: runner.pipelines[pipelineName].pipeline.Path,
			}

			if !stack.deployed.Swap(true) {
				runner.event(interactive.SetStateEvent{
					Name:  "stack/" + stack.name.DNSName(),
					State: interactive.Started,
					Stage: "deploying",
				})
				result := EnvironmentSetupResult{
					EnvironmentInfo: info,
					SetupType:       EnvironmentDeployment,
					Started:         time.Now(),
				}
				pipeline, err := runner.pipeline(stack.pipelineName)
				result.Completed = time.Now()
				result.Err = errToStringPtr(err)
				runner.options.Reporter.EnvironmentSetupResult(&result)
				if err != nil {
					return err
				}
				if _, err := deploy.Exec(gctx, cfg, pipeline.pipeline, stack.name, k8sClient); err != nil {
					return fmt.Errorf("deploy failed for stack %s: %v", stack.name, err)
				}
				close(stack.deployedc)
			}

			select {
			case <-gctx.Done():
				return gctx.Err()
			case <-stack.deployedc:
			}

			if !stack.before.Swap(true) {
				runner.event(interactive.SetStateEvent{
					Name:  "stack/" + stack.name.DNSName(),
					State: interactive.Started,
					Stage: "initializing",
				})
				result := EnvironmentSetupResult{
					EnvironmentInfo: info,
					SetupType:       EnvironmentInitialization,
					Started:         time.Now(),
				}
				pipeline, err := runner.pipeline(stack.pipelineName)
				result.Completed = time.Now()
				result.Err = errToStringPtr(err)
				runner.options.Reporter.EnvironmentSetupResult(&result)
				if err != nil {
					return err
				}
				if pipeline.batchPipeline.Setup != "" {
					s, err := pipeline.pipeline.Setups.Get(pipeline.batchPipeline.Setup)
					if err != nil {
						return err
					}
					err = run.ExecHooks(
						ctx,
						cfg,
						stack.name,
						"before",
						s.Before,
						nil,
						runner.k8sClient)
					if err != nil {
						return err
					}
				}
				close(stack.initializedc)
			}

			select {
			case <-gctx.Done():
				return gctx.Err()
			case <-stack.initializedc:
			}

			runner.event(interactive.SetStateEvent{
				Name:  "stack/" + stack.name.DNSName(),
				State: interactive.Completed,
			})

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

	stackCtx, cancelDetached := context.WithCancel(ctx)
	defer func() {
		cancelDetached()
	}()
	{
		for _, stack := range stacks {
			stack := stack
			go func() {
				select {
				case <-stackCtx.Done():
					return
				case err := <-stack.holdErrc:
					if err != nil {
						runner.cfg.Logger().Error(logDomain, "detached error: %v", err)
						cancelDetached()
					}
				}
			}()
		}
	}

	// Gather environment variables
	runner.event(interactive.SetStateEvent{
		Name:  cmd.Name,
		State: interactive.Started,
		Stage: "environment",
	})
	var allEnv []string
	var allEnvMut sync.Mutex
	g, gctx = errgroup.WithContext(stackCtx)
	for _, stack := range stacks {
		stack := stack
		g.Go(func() error {
			pipeline, err := runner.pipeline(stack.pipelineName)
			if err != nil {
				return err
			}

			genv, genvctx := errgroup.WithContext(gctx)
			if pipeline.batchPipeline.Setup != "" {
				setup, err := pipeline.pipeline.Setups.Get(pipeline.batchPipeline.Setup)
				if err != nil {
					return err
				}
				for _, e := range setup.Env {
					e := e
					genv.Go(func() error {
						s, err := runner.transGet(genvctx, stack.name, e)
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
	allEnv = append(allEnv, runner.sharedEnv...)

	var maxTries int
	if cmd.Flaky && !runner.options.Bail {
		maxTries = 3
	} else {
		maxTries = 1
	}

	var err error
	tries := 1
	for {
		stage := ""
		if tries > 1 {
			stage = fmt.Sprintf("retry %d/%d", tries-1, maxTries-1)
		}
		runner.event(interactive.SetStateEvent{
			Name:  cmd.Name,
			State: interactive.Started,
			Stage: stage,
		})

		info := CommandInfo{
			BatchID: runner.batchID,
			Name:    cmd.Name,
			Tries:   tries,
		}

		procCmd := proc.GracefulCommandContext(stackCtx, cmd.Command[0], cmd.Command[1:]...)
		if cmd.WorkingDir != "" {
			procCmd.Dir = cfg.Path(cmd.WorkingDir)
		}

		procCmd.Env = append(os.Environ(), allEnv...)

		scannerDone := make(chan struct{})
		{
			stdout, err := procCmd.StdoutPipe()
			if err != nil {
				// This means that Pipe was invoked on a cmd that has either
				// its os.Stdout already set, or has already been started.
				// Here, that is a logic error.
				panic(fmt.Errorf("could not pipe command stdout: %v", err))
			}
			stderr, err := procCmd.StderrPipe()
			if err != nil {
				// This means that Pipe was invoked on a cmd that has either
				// its os.Stdout already set, or has already been started.
				// Here, that is a logic error.
				panic(fmt.Errorf("could not pipe command stderr: %v", err))
			}
			combinedOutput := io.MultiReader(stdout, stderr)
			go func() {
				w, err := runner.options.Reporter.CommandOutput(&info)
				if err != nil {
					cfg.Logger().Error("run:"+cmd.Name, "%v", err)
					return
				}
				defer w.Close()
				defer close(scannerDone)
				scanner := bufio.NewScanner(combinedOutput)
				for scanner.Scan() {
					cfg.Logger().Info("run:"+cmd.Name, "%s", scanner.Text())
					if _, err := w.Write(append(scanner.Bytes(), '\n')); err != nil {
						cfg.Logger().Error("run:"+cmd.Name, "could not write to log file: %v", err)
						return
					}
				}
			}()
		}

		result := CommandResult{
			CommandInfo: info,
			Started:     time.Now(),
		}
		err = procCmd.Start()
		<-scannerDone
		if err == nil {
			err = procCmd.Wait()
		}
		result.Completed = time.Now()

		if err == nil {
			cfg.Logger().Info("run:"+cmd.Name, "SUCCESS")
			runner.options.Reporter.CommandResult(&result)
			break
		}
		cfg.Logger().Error("run:"+cmd.Name, "%v", err)
		result.Err = errToStringPtr(err)
		runner.options.Reporter.CommandResult(&result)
		if tries == maxTries {
			break
		}
		tries++
	}

	if err != nil {
		if runner.options.Bail {
			return fmt.Errorf("could not run '%s': %v", cmd.Name, err)
		}
		runner.erroredMut.Lock()
		runner.errored = append(runner.errored, cmd.Name)
		runner.erroredMut.Unlock()
	}
	runner.event(interactive.SetStateEvent{
		Name:  cmd.Name,
		State: interactive.Completed,
	})
	return nil
}

func (runner *runner) transGet(ctx context.Context, name names.Name, tplStr string) (string, error) {
	runner.transMut.Lock()
	trans, ok := runner.trans[name.DNSName()]
	if !ok {
		trans = env.NewTransformer(env.K8sTemplateFuncs(runner.cfg, name, runner.k8sClient))
	}
	runner.transMut.Unlock()

	return trans.Get(ctx, tplStr)
}

func (runner *runner) pipeline(name string) (*pipeline, error) {
	pipeline, ok := runner.pipelines[name]
	if !ok {
		return nil, fmt.Errorf("cannot find pipeline with name '%s'", name)
	}
	return pipeline, nil
}

func (runner *runner) event(event interface{}) {
	if runner.options.Events != nil {
		runner.options.Events <- event
	} else {
		runner.cfg.Logger().Info(logDomain, "%v", event)
	}
}

func errToStringPtr(err error) *string {
	if err == nil {
		return nil
	}
	s := err.Error()
	return &s
}
