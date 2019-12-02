// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package config

import (
	"bytes"
	"fmt"
	"github.com/Masterminds/sprig"
	"github.com/go-playground/validator"
	"github.com/hchauvin/warp/pkg/templates"
	"github.com/pelletier/go-toml"
	"github.com/spf13/afero"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
)

// Read reads the project-wide configuration.
func Read(path string) (*Config, error) {
	return ReadFs(afero.NewOsFs(), path)
}

// ReadFs does the same as Read but on an arbitrary afero file system.
func ReadFs(fs afero.Fs, path string) (*Config, error) {
	b, err := afero.ReadFile(fs, path)
	if err != nil {
		return nil, fmt.Errorf("could not read config file '%s': %v", path, err)
	}

	root, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return nil, err
	}

	tpl, err := template.New("config").
		Funcs(sprig.TxtFuncMap()).
		Funcs(templates.TxtFuncMap()).
		Parse(string(b))
	if err != nil {
		return nil, fmt.Errorf("cannot parse template: %v", err)
	}
	data := map[string]interface{}{
		"Root": root,
	}
	w := &bytes.Buffer{}
	if err := tpl.Execute(w, data); err != nil {
		return nil, fmt.Errorf("cannot expand template: %v", err)
	}

	cfg := &Config{}
	if err := toml.Unmarshal(w.Bytes(), cfg); err != nil {
		return nil, fmt.Errorf("cannot read config: %v", err)
	}

	if err := validator.New().Struct(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %v", err)
	}

	if cfg.Tools == nil {
		cfg.Tools = make(map[Tool]ToolInfo)
	}
	for _, toolName := range ToolNames {
		tool, _ := cfg.Tools[toolName]
		if tool.Path == "" {
			tool.Path = toolDefaultPaths[toolName]
		}
		cfg.Tools[toolName] = tool
	}

	if cfg.Kubernetes != nil && len(cfg.Kubernetes.Kubeconfig) != 0 {
		var envVar strings.Builder
		for i, file := range cfg.Kubernetes.Kubeconfig {
			expanded, err := expandPath(file)
			if err != nil {
				return nil, err
			}
			if !filepath.IsAbs(expanded) {
				expanded = filepath.Join(root, expanded)
			}
			if i > 0 {
				if runtime.GOOS == "windows" {
					envVar.WriteRune(';')
				} else {
					envVar.WriteRune(':')
				}
			}
			envVar.WriteString(expanded)
		}
		cfg.Kubernetes.KubeconfigEnvVar = envVar.String()
	}

	cfg.WorkspaceDir = root

	return cfg, nil
}

func expandPath(path string) (string, error) {
	if !strings.HasPrefix(path, "~/") {
		return path, nil
	}
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	dir := usr.HomeDir
	return filepath.Join(dir, path[2:]), nil
}
