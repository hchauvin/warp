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
	"github.com/hchauvin/warp/pkg/k8s"
	"github.com/hchauvin/warp/pkg/pipelines"
	"github.com/hchauvin/warp/pkg/stacks"
	"github.com/hchauvin/warp/pkg/stacks/names"
	run_batch "github.com/hchauvin/warp/pkg/run/batch"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
)

const logDomain = "warp"

// HoldConfig gives the configuration for the Hold function.
type HoldConfig struct {
	WorkingDir   string
	ConfigPath   string
	PipelinePath string
	Dev          bool
	Tail         bool
	Run          []string
	Wait         bool
	Rm           bool
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

	var expandedPipelineFolder string
	if pipeline.Stack.Name != "" {
		expandedPipelineFolder = pipeline.Stack.Name
	} else if pipeline.Stack.Family != "" {
		expandedPipelineFolder = pipeline.Stack.Family
	} else {
		return errors.New("invalid pipeline: neither stack.name nor stack.family is given")
	}
	expandedPipelineFolder = cfg.Path(filepath.Join(cfg.OutputRoot, "pipelines", expandedPipelineFolder))
	if err := os.MkdirAll(expandedPipelineFolder, 0777); err != nil {
		return err
	}
	pipelineYaml, err := yaml.Marshal(pipeline)
	if err != nil {
		return err
	}
	pipelinePath := filepath.Join(expandedPipelineFolder, "expanded_pipeline.yml")
	if err := ioutil.WriteFile(
		pipelinePath,
		pipelineYaml,
		0777); err != nil {
		return err
	}
	cfg.Logger().Info("pipelines", "pipeline expanded to '%s'", pipelinePath)

	name, releaseName, err := stacks.Hold(cfg, pipeline)
	if err != nil {
		return err
	}
	defer releaseName()

	var errs []string

	stacksExecCtx, cancelStacksExec := context.WithCancel(context.Background())
	detachedErrc := make(chan error, 1)
	go func() {
		select {
		case err := <-detachedErrc:
			if err != context.Canceled {
				cancelStacksExec()
				errs = append(errs, err.Error())
			}
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
			WaitForInterrupt: len(holdCfg.Run) == 0 || holdCfg.Wait,
		}, detachedErrc)
		if err != nil && err != context.Canceled {
			errs = append(errs, err.Error())
		}
	}

	if holdCfg.Rm {
		if err := stacks.Remove(context.Background(), cfg, pipeline, name.ShortName); err != nil {
			errs = append(errs, fmt.Errorf("while cleaning: %v", err).Error())
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

// BatchCfg configures the Batch function.
type BatchCfg struct {
	WorkingDir   string
	ConfigPath   string
	BatchPath string
	Parallelism int
	MaxStacksPerPipeline int
	Tags string
	Focus string
	Bail bool
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

	return run_batch.RunBatch(ctx, cfg, filteredBatch, &run_batch.RunBatchOptions{
		Parallelism: batchCfg.Parallelism,
		MaxStacksPerPipeline: batchCfg.MaxStacksPerPipeline,
		Bail: batchCfg.Bail,
	}, k8sClient)
}

// RmCfg configures the Rm function.
type RmCfg struct {
	WorkingDir   string
	ConfigPath   string
	PipelinePath string
	ShortNames   []string
	All          bool
}

// Rm removes a stack.
func Rm(rmCfg *RmCfg) error {
	cfg, err := readConfig(rmCfg.WorkingDir, rmCfg.ConfigPath)
	if err != nil {
		return err
	}

	pipeline, err := pipelines.Read(cfg, rmCfg.PipelinePath)
	if err != nil {
		return err
	}

	shortNames := rmCfg.ShortNames
	if len(shortNames) == 0 {
		shortNames, err = stacks.List(context.Background(), cfg, pipeline, !rmCfg.All)
		if err != nil {
			return err
		}
	}

	cfg.Logger().Info(logDomain, "removing stacks: %s", strings.Join(shortNames, " "))

	var g errgroup.Group
	for _, shortName := range shortNames {
		shortName := shortName
		g.Go(func() error {
			return stacks.Remove(context.Background(), cfg, pipeline, shortName)
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

type GcCfg struct {
	WorkingDir string
	ConfigPath string
	Family     string
}

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

	sem := semaphore.NewWeighted(10) // 10 is a sensible default
	g, ctx := errgroup.WithContext(ctx)
	for _, name := range nameList {
		if gcCfg.Family != "" && name.Family != "" {
			continue
		}
		if !name.Free {
			continue
		}
		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}
		name := name
		g.Go(func() error {
			defer sem.Release(1)

			if err := nameManager.TryAcquire(name.Family, name.Name); err != nil {
				return nil // Cannot acquire, skip garbage collection
			}
			defer nameManager.Release(name.Family, name.Name)

			cfg.Logger().Info(logDomain+":gc", "family=%s shortName=%s", name.Family, name.Name)
			err := k8sClient.Gc(ctx, cfg, names.Name{Family: name.Family, ShortName: name.Name})
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
