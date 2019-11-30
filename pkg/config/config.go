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

// Config is the project-wide configuration for warp.
type Config struct {
	// NameManagerURL is the URL for the name_manager backend.
	// name_manager is used to acquire/release stacks with no
	// race condition.
	//
	// See https://github.com/hchauvin/name_manager.
	NameManagerURL string

	// OutputRoot is the path to a folder, given relative to
	// the parent folder of the Config file, where all the
	// intermediary artifacts are put.  This folder is
	// within the workspace to allow for easy inspection
	// during debugging.
	OutputRoot string

	// Tools configures tools.  By default, the tools are looked up
	// in the PATH folders.
	Tools map[Tool]*ToolInfo

	// WorkspaceDir is the workspace directory.
	WorkspaceDir string `toml:"-"`
}

// Path resolves a path relative to the workspace dir.
func (cfg *Config) Path(path string) string {
	return filepath.Join(cfg.WorkspaceDir, path)
}

// Logger gives the logger associated with this configuration.
func (cfg *Config) Logger() *log.Logger {
	return &log.Logger{}
}

// Tool is the name of a tool.
type Tool string

const (
	Kustomize   = Tool("kustomize")
	Kubectl     = Tool("kubectl")
	Ksync       = Tool("ksync")
	BrowserSync = Tool("browserSync")
	Docker      = Tool("docker")
)

// ToolNames gives all the required tools.
var ToolNames = []Tool{Kustomize, Kubectl, Ksync, BrowserSync, Docker}

// LogDomain gives the log domain for a tool.
func (tool Tool) LogDomain() string {
	return "tool." + toolDefaultPaths[tool]
}

// ToolInfo configures a tool.
type ToolInfo struct {
	// Path is the path to the tool on the local file system.
	Path string
}

func (ti *ToolInfo) Resolve() (fullPath string, err error) {
	return ti.Path, nil
}

var toolDefaultPaths = map[Tool]string{
	Kustomize:   "kustomize",
	Kubectl:     "kubectl",
	Ksync:       "ksync",
	BrowserSync: "browser-sync",
	Docker:      "docker",
}
