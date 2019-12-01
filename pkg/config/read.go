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
	"path/filepath"
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
		cfg.Tools = make(map[Tool]*ToolInfo)
	}
	for _, toolName := range ToolNames {
		var tool *ToolInfo
		tool, ok := cfg.Tools[toolName]
		if !ok {
			tool = &ToolInfo{}
		}
		if tool.Path == "" {
			tool.Path = toolDefaultPaths[toolName]
		}
		cfg.Tools[toolName] = tool
	}

	cfg.WorkspaceDir = root

	return cfg, nil
}
