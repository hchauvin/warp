// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

// Package config provides TOML-based configuration for warp (.warprc.toml).  Used
// to set up paths to tools, amongst other things.
package config

import (
	"fmt"
	"github.com/hchauvin/warp/pkg/log"
	"path/filepath"
	"strings"
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
	Tools map[Tool]ToolInfo

	// Kubernetes holds the (default) configuration for the Kubernetes
	// cluster used for deployment.
	Kubernetes *Kubernetes

	Telemetry Telemetry

	// WorkspaceDir is the workspace directory.
	WorkspaceDir string `toml:"-"`

	// RunID is a random ID specific to this run of "warp".
	RunID string `toml:"-"`

	logger log.Logger
}

// Path resolves a path relative to the workspace dir.
func (cfg *Config) Path(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(cfg.WorkspaceDir, path)
}

// Logger gives the logger associated with this configuration.
func (cfg *Config) Logger() *log.Logger {
	return &cfg.logger
}

// ToolPath resolves the full path of a tool.  It errors if
// the tool could not be found.
func (cfg *Config) ToolPath(tool Tool) (fullPath string, err error) {
	path := cfg.Tools[tool].Path
	if path == "" {
		panic(fmt.Sprintf("unexpected empty path for tool %s", tool))
	}
	if filepath.IsAbs(path) {
		return path, nil
	}
	if !strings.Contains(path, "/") {
		return path, nil
	}
	if strings.HasPrefix(path, "./") {
		path = path[2:]
	}
	return filepath.Join(cfg.WorkspaceDir, path), nil
}

// Tool is the name of a tool.
type Tool string

// Names of the tools used by warp.
const (
	Kustomize   = Tool("Kustomize")
	Helm        = Tool("Helm")
	KubeScore   = Tool("KubeScore")
	Kubectl     = Tool("Kubectl")
	Ksync       = Tool("Ksync")
	BrowserSync = Tool("BrowserSync")
	Docker      = Tool("Docker")
)

// ToolNames gives all the required tools.
var ToolNames = []Tool{Kustomize, Helm, KubeScore, Kubectl, Ksync, BrowserSync, Docker}

// LogDomain gives the log domain for a tool.
func (tool Tool) LogDomain() string {
	return "tool." + toolDefaultPaths[tool]
}

// ToolInfo configures a tool.
type ToolInfo struct {
	// Path is the path to the tool on the local file system.
	Path string
}

var toolDefaultPaths = map[Tool]string{
	Kustomize:   "kustomize",
	Helm:        "helm",
	KubeScore:   "kube-score",
	Kubectl:     "kubectl",
	Ksync:       "ksync",
	BrowserSync: "browser-sync",
	Docker:      "docker",
}

// Kubernetes holds the configuration for a Kubernetes
// cluster used for deployment.
type Kubernetes struct {
	// DefaultContext is the default kubeconfig context to use.  If omitted,
	// the current kubeconfig context is used.
	DefaultContext string

	// Kubeconfig is a list of configuraton files that are merged
	// to give the final kubeconfig configuration.  The files
	// are "~" (home) expanded.
	//
	// For background info on configuration merging, see
	// https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/#the-kubeconfig-environment-variable
	Kubeconfig []string

	// Resources is a list of custom resources to be managed by warp,
	// especially during garbage collection.
	Resources []Resource

	// PreservePVCByDefault should be set to true to enable preserving
	// persistent volume claims by default during garbage collection.
	PreservePVCByDefault bool

	// KubeconfigEnvVar is the value of the KUBECONFIG environment
	// variable that is created from Kubeconfig.
	KubeconfigEnvVar string `toml:"-"`
}

// Resource identifies custom resources by group, version, and kind.
type Resource struct {
	Group    string
	Version  string
	Resource string
}

type Telemetry struct {
	// Connection string for the telemetry module
	ConnectionString string
}
