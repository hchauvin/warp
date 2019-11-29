// config provides TOML-based configuration for warp (.warprc.toml).  Used
// to set up paths to tools, amongst other things.
//
// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package config

import (
	"github.com/hchauvin/warp/pkg/log"
	"path/filepath"
)

type Config struct {
	NameManagerURL string
	OutputRoot     string
	Tools          map[Tool]*ToolInfo

	workspaceDir string
}

func (cfg *Config) WorkspaceDir() string {
	return cfg.workspaceDir
}

func (cfg *Config) SetWorkspaceDir(dir string) {
	cfg.workspaceDir = dir
}

func (cfg *Config) Path(path string) string {
	return filepath.Join(cfg.workspaceDir, path)
}

func (cfg *Config) Logger() *log.Logger {
	return &log.Logger{}
}

type Tool string

const (
	Kustomize   = Tool("kustomize")
	Kubectl     = Tool("kubectl")
	Ksync       = Tool("ksync")
	BrowserSync = Tool("browserSync")
	Docker      = Tool("docker")
)

var ToolNames = []Tool{Kustomize, Kubectl, Ksync, BrowserSync, Docker}

func (tool Tool) LogDomain() string {
	return "tool." + toolDefaultPaths[tool]
}

type ToolInfo struct {
	Path       string
	MinVersion string

	fullPath string
}

func (ti *ToolInfo) Resolve() (fullPath string, err error) {
	if ti.fullPath == "" {
		// TODO: Windows + version check
		ti.fullPath = ti.Path
	}
	return ti.fullPath, nil
}

var toolDefaultPaths = map[Tool]string{
	Kustomize:   "kustomize",
	Kubectl:     "kubectl",
	Ksync:       "ksync",
	BrowserSync: "browser-sync",
	Docker:      "docker",
}

/*var toolMinVersions = map[Tool]string{
	Kustomize: "3.4.0",
	Kubectl: "1.16.3",
	Ksync: "0.4.1",
	BrowserSync: "2.26.7",
}*/
