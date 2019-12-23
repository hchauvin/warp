// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package warp

import (
	"context"
	"errors"
	"fmt"
	"github.com/hchauvin/name_manager/pkg/name_manager"
	"github.com/hchauvin/warp/pkg/batches"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/deploy"
	"github.com/hchauvin/warp/pkg/k8s"
	"github.com/hchauvin/warp/pkg/lint"
	"github.com/hchauvin/warp/pkg/log/interactive"
	"github.com/hchauvin/warp/pkg/pipelines"
	run_batch "github.com/hchauvin/warp/pkg/run/batch"
	"github.com/hchauvin/warp/pkg/run/batch/fsreporter"
	"github.com/hchauvin/warp/pkg/stacks"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
)

const logDomain = "warp"

// LintCfg configures the "lint" command.
type LintCfg struct {
	WorkingDir   string
	ConfigPath   string
	PipelinePath string
}

// Lint implements the "lint" command.
func Lint(ctx context.Context, lintCfg *LintCfg) error {
	cfg, err := readConfig(lintCfg.WorkingDir, lintCfg.ConfigPath)
	if err != nil {
		return err
	}

	pipeline, err := pipelines.Read(cfg, lintCfg.PipelinePath)
	if err != nil {
		return err
	}

	if err := pipeline.Expand(cfg); err != nil {
		return err
	}

	return lint.Lint(ctx, cfg, pipeline)
}

// HoldConfig gives the configuration for the Hold function.
type HoldConfig struct {
	WorkingDir   string
	ConfigPath   string
	PipelinePath string
	Dev          bool
	Tail         bool
	Run          []string
	Setup        string
	DumpEnv      string
	PersistEnv   bool
	Wait         bool
}

// Hold deploy a stacks, then hold it until either 1) the run specifications
// are executed, 2) the user requests it (via, e.g., Ctl-C).
func Hold(holdCfg *HoldConfig) error {
	cfg, err := readConfig(holdCfg.WorkingDir, holdCfg.ConfigPath)
	if err != nil {
		return err
	}

	pipeline, err := pipelines.Read(cfg, holdCfg.PipelinePath)
	if err != nil {
		return err
	}

	if err := pipeline.Expand(cfg); err != nil {
		return err
	}

	name, holdErrc, releaseName, err := stacks.Hold(cfg, pipeline)
	if err != nil {
		return err
	}
	defer releaseName()

	var errs []string

	stacksExecCtx, cancelStacksExec := context.WithCancel(context.Background())
	detachedErrc := make(chan error, 1)
	go func() {
		var err error
		select {
		case err = <-detachedErrc:
		case err = <-holdErrc:
		}
		if err != nil && err != context.Canceled {
			cancelStacksExec()
			errs = append(errs, err.Error())
		}
	}()
	signalc := make(chan os.Signal)
	signal.Notify(signalc, os.Interrupt)
	select {
	case <-signalc:
		cfg.Logger().Info(logDomain, "cleaning up...")
		cancelStacksExec()
	case err := <-detachedErrc:
		errs = append(errs, err.Error())
	default:
		err := stacks.Exec(stacksExecCtx, cfg, pipeline, &stacks.ExecConfig{
			Name:             *name,
			Dev:              holdCfg.Dev,
			Tail:             holdCfg.Tail,
			Run:              holdCfg.Run,
			Setup:            holdCfg.Setup,
			DumpEnv:          holdCfg.DumpEnv,
			PersistEnv:       holdCfg.PersistEnv,
			WaitForInterrupt: len(holdCfg.Run) == 0 || holdCfg.Wait,
		}, detachedErrc)
		if err != nil && err != context.Canceled {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

// DeployCfg configures the "deploy" command.
type DeployCfg struct {
	WorkingDir   string
	ConfigPath   string
	PipelinePath string
}

// Deploy implements the "deploy" command.
func Deploy(ctx context.Context, deployCfg *DeployCfg) error {
	cfg, err := readConfig(deployCfg.WorkingDir, deployCfg.ConfigPath)
	if err != nil {
		return err
	}

	pipeline, err := pipelines.Read(cfg, deployCfg.PipelinePath)
	if err != nil {
		return err
	}

	if pipeline.Stack.Name == "" {
		return errors.New("cannot deploy a nameless stack")
	}

	if err := pipeline.Expand(cfg); err != nil {
		return err
	}

	k8sClient, err := k8s.New(cfg)
	if err != nil {
		return err
	}
	defer k8sClient.Ports.CancelForwarding()

	if err := deploy.Exec(ctx, cfg, pipeline, names.Name{ShortName: pipeline.Stack.Name}, k8sClient); err != nil {
		return fmt.Errorf("deploy step failed: %v", err)
	}

	return nil
}

// BatchCfg configures the Batch function.
type BatchCfg struct {
	WorkingDir           string
	ConfigPath           string
	BatchPath            string
	Parallelism          int
	MaxStacksPerPipeline int
	Tags                 string
	Focus                string
	Bail                 bool
	Advisory             bool
	Report               string
	Stream               bool
}

// Batch executes a batch.
func Batch(ctx context.Context, batchCfg *BatchCfg) error {
	cfg, err := readConfig(batchCfg.WorkingDir, batchCfg.ConfigPath)
	if err != nil {
		return err
	}

	batch, err := batches.Read(cfg, batchCfg.BatchPath)
	if err != nil {
		return err
	}

	filteredBatch, err := batch.Filter(batchCfg.Tags, batchCfg.Focus)
	if err != nil {
		return err
	}

	k8sClient, err := k8s.New(cfg)
	if err != nil {
		return err
	}
	defer k8sClient.Ports.CancelForwarding()

	var reporter run_batch.Reporter
	if batchCfg.Report == "" {
		reporter = &run_batch.NoopReporter{}
	} else {
		reporter, err = fsreporter.New(batchCfg.Report)
		if err != nil {
			return err
		}
	}

	var events chan interface{}
	runBatchDone := make(chan struct{})
	var interactiveReportDone chan struct{}
	if !batchCfg.Stream {
		events = make(chan interface{})
		interactiveReportDone = make(chan struct{})
		go func() {
			defer func() {
				interactiveReportDone <- struct{}{}
			}()
			if err := interactive.Report(cfg.Logger(), events, runBatchDone); err != nil {
				cfg.Logger().Error("interactive", "%v", err)
			}
		}()
	}
	err = run_batch.RunBatch(ctx, cfg, filteredBatch, &run_batch.RunBatchOptions{
		Parallelism:          batchCfg.Parallelism,
		MaxStacksPerPipeline: batchCfg.MaxStacksPerPipeline,
		Bail:                 batchCfg.Bail,
		Advisory:             batchCfg.Advisory,
		Reporter:             reporter,
		Events:               events,
	}, k8sClient)
	close(runBatchDone)
	if interactiveReportDone != nil {
		<-interactiveReportDone
	}
	return err
}

// GcCfg configures the "gc" command.
type GcCfg struct {
	WorkingDir                     string
	ConfigPath                     string
	Family                         string
	PreservePersistentVolumeClaims bool
	DiscardPersistentVolumeClaims  bool
}

// Gc implements the "gc" command.
func Gc(ctx context.Context, gcCfg *GcCfg) error {
	cfg, err := readConfig(gcCfg.WorkingDir, gcCfg.ConfigPath)
	if err != nil {
		return err
	}

	k8sClient, err := k8s.New(cfg)
	if err != nil {
		return err
	}
	defer k8sClient.Ports.CancelForwarding()

	nameManager, err := name_manager.CreateFromURL(cfg.NameManagerURL)
	if err != nil {
		return fmt.Errorf("cannot create name manager: %v", err)
	}

	nameList, err := nameManager.List()
	if err != nil {
		return err
	}

	if gcCfg.PreservePersistentVolumeClaims && gcCfg.DiscardPersistentVolumeClaims {
		return fmt.Errorf("--preserve_pvc and --discard_pvc cannot be both present")
	}
	preservePersistentVolumeClaims := cfg.Kubernetes.PreservePVCByDefault
	if gcCfg.PreservePersistentVolumeClaims {
		preservePersistentVolumeClaims = true
	}
	if gcCfg.DiscardPersistentVolumeClaims {
		preservePersistentVolumeClaims = false
	}

	sem := semaphore.NewWeighted(10) // 10 is a sensible default
	g, ctx := errgroup.WithContext(ctx)
	for _, name := range nameList {
		if gcCfg.Family != "" && name.Family != "" {
			continue
		}
		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}
		name := name
		g.Go(func() error {
			defer sem.Release(1)

			if err := nameManager.TryAcquire(name.Family, name.Name); err != nil {
				cfg.Logger().Info(logDomain+":gc", "BUSY: family=%s shortName=%s", name.Family, name.Name)
				return nil // Cannot acquire, skip garbage collection
			}
			defer nameManager.Release(name.Family, name.Name)

			cfg.Logger().Info(logDomain+":gc", "PENDING: family=%s shortName=%s", name.Family, name.Name)
			err := k8sClient.Gc(
				ctx,
				cfg,
				names.Name{Family: name.Family, ShortName: name.Name},
				&k8s.GcOptions{
					PreservePersistentVolumeClaims: preservePersistentVolumeClaims,
				})
			if err != nil {
				return err
			}
			cfg.Logger().Info(logDomain+":gc", "DONE: family=%s shortName=%s", name.Family, name.Name)
			return nil
		})
	}

	return g.Wait()
}

func readConfig(workingDir, configPath string) (*config.Config, error) {
	fullPath := filepath.Join(workingDir, configPath)
	return config.Read(fullPath)
}
