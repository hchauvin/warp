// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package warp

import (
	"context"
	"errors"
	"fmt"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/pipelines"
	"github.com/hchauvin/warp/pkg/stacks"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
)

const logDomain = "warp"

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

type RmCfg struct {
	WorkingDir   string
	ConfigPath   string
	PipelinePath string
	ShortNames   []string
	All          bool
}

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

/*type GcCfg struct {
	WorkingDir string
	ConfigPath string
	Family string
	All bool
}

func Gc(gcCfg *GcCfg) error {
	cfg, err := readConfig(gcCfg.WorkingDir, gcCfg.ConfigPath)
	if err != nil {
		return err
	}

	nameManager, err := name_manager.CreateFromURL(cfg.NameManagerURL)
	if err != nil {
		return fmt.Errorf("cannot create name manager: %v", err)
	}
	families, err := listFamilyNames(nameManager)
	if err != nil {
		return err
	}


}*/

func readConfig(workingDir, configPath string) (*config.Config, error) {
	fullPath := filepath.Join(workingDir, configPath)
	return config.Read(fullPath)
}
