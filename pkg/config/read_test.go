// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package config

import (
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRead(t *testing.T) {
	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, "/workspace/config.yml", simpleConfigBytes, 0666)
	assert.NoError(t, err)

	cfg, err := ReadFs(fs, "/workspace/config.yml")
	assert.NoError(t, err)

	if runtime.GOOS == "windows" {
		assert.Equal(t, filepath.FromSlash("C:/workspace"), cfg.WorkspaceDir)
	} else {
		assert.Equal(t, filepath.FromSlash("/workspace"), cfg.WorkspaceDir)
	}
	assert.Equal(t, "mongo://xxx", cfg.NameManagerURL)
	assert.Equal(t, ".warp", cfg.OutputRoot)
}

func TestReadKubeconfig(t *testing.T) {
	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, "/workspace/config.yml", configWithKubeconfigBytes, 0666)
	assert.NoError(t, err)

	cfg, err := ReadFs(fs, "/workspace/config.yml")
	assert.NoError(t, err)

	assert.NotNil(t, cfg.Kubernetes)

	usr, err := user.Current()
	if err != nil {
		assert.FailNow(t, "cannot get OS user", usr)
	}
	components := []string{
		filepath.Join(usr.HomeDir, "in-home"),
		filepath.FromSlash("/workspace/relative"),
		filepath.FromSlash("/absolute"),
	}
	assert.Equal(t, strings.Join(components, string(os.PathListSeparator)), cfg.Kubernetes.KubeconfigEnvVar)
}

var simpleConfigBytes = []byte(`
NameManagerURL = "mongo://xxx"
OutputRoot = ".warp"
`)

var configWithKubeconfigBytes = []byte(`
NameManagerURL = "mongo://xxx"
OutputRoot = ".warp"

[Kubernetes]
Kubeconfig = ["~/in-home", "relative", "/absolute"]
`)
