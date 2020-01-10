// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package warp

import (
	"context"
	_ "github.com/hchauvin/name_manager/pkg/local_backend"
	"github.com/hchauvin/name_manager/pkg/name_manager"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/k8s"
	"github.com/hchauvin/warp/pkg/pipelines"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"github.com/pelletier/go-toml"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestDeploy(t *testing.T) {
	dir, err := ioutil.TempDir("", "warp_gc")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	// Write workspace
	cfgTOML, err := toml.Marshal(config.Config{})
	assert.NoError(t, err)

	err = ioutil.WriteFile(filepath.Join(dir, ".warprc.toml"), cfgTOML, 0666)
	assert.NoError(t, err)

	pipelineYAML, err := yaml.Marshal(pipelines.Pipeline{
		Stack: pipelines.Stack{
			Name: "foo",
		},
	})
	assert.NoError(t, err)

	err = ioutil.WriteFile(filepath.Join(dir, "pipeline.yaml"), pipelineYAML, 0666)
	assert.NoError(t, err)

	execCalled := false
	exec := func(ctx context.Context, cfg *config.Config, pipeline *pipelines.Pipeline, name names.Name, k8sClient *k8s.K8s) error {
		execCalled = true
		assert.Equal(t, names.Name{ShortName: "foo"}, name)
		assert.Equal(t, "foo", pipeline.Stack.Name)
		return nil
	}
	err = doDeploy(context.Background(), &DeployCfg{
		WorkingDir:   dir,
		ConfigPath:   ".warprc.toml",
		PipelinePath: "pipeline.yaml",
	}, exec)
	assert.NoError(t, err)
	assert.True(t, execCalled)
}

func TestGc(t *testing.T) {
	dir, err := ioutil.TempDir("", "warp_gc")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	nmURL := "local://" + filepath.Join(dir, "name_manager.db")

	nm, err := name_manager.CreateFromURL(nmURL)
	assert.NoError(t, err)

	// Sets up the names
	name, err := nm.Acquire("foo")
	assert.NoError(t, err)
	assert.Equal(t, "0", name)

	name, err = nm.Acquire("foo")
	assert.NoError(t, err)
	assert.Equal(t, "1", name)

	err = nm.Release("foo", "0")
	assert.NoError(t, err)

	name, err = nm.Acquire("bar")
	assert.NoError(t, err)
	assert.Equal(t, "0", name)

	err = nm.Release("bar", "0")
	assert.NoError(t, err)

	// Checks that we've got the right status for the names
	nameList, err := nm.List()
	namesFree := make(map[string]bool, len(nameList))
	for _, n := range nameList {
		namesFree[n.Family+"_"+n.Name] = n.Free
	}
	assert.Equal(
		t,
		map[string]bool{
			"foo_0": true,
			"foo_1": false,
			"bar_0": true,
		},
		namesFree)

	// Write configuration
	cfgTOML, err := toml.Marshal(config.Config{
		NameManagerURL: nmURL,
	})
	assert.NoError(t, err)

	err = ioutil.WriteFile(filepath.Join(dir, ".warprc.toml"), cfgTOML, 0666)
	assert.NoError(t, err)

	// Execute garbage collection
	gcClient := &mockGcClient{}
	err = gc(context.Background(), &GcCfg{
		WorkingDir:                     dir,
		ConfigPath:                     ".warprc.toml",
		Family:                         "foo",
		PreservePersistentVolumeClaims: false,
		DiscardPersistentVolumeClaims:  false,
	}, gcClient)
	assert.NoError(t, err)

	// Check that the right stacks have been garbage-collected
	assert.ElementsMatch(
		t,
		[]gcCall{
			{
				name: names.Name{Family: "foo", ShortName: "0"},
				options: &k8s.GcOptions{
					PreservePersistentVolumeClaims: false,
				},
			},
		},
		gcClient.calls)
}

func TestGetPreservePersistentVolumeClaims(t *testing.T) {
	b, err := getPreservePersistentVolumeClaims(
		&config.Config{},
		&GcCfg{})
	assert.NoError(t, err)
	assert.False(t, b)

	b, err = getPreservePersistentVolumeClaims(
		&config.Config{},
		&GcCfg{
			PreservePersistentVolumeClaims: true,
		})
	assert.NoError(t, err)
	assert.True(t, b)

	b, err = getPreservePersistentVolumeClaims(
		&config.Config{
			Kubernetes: &config.Kubernetes{
				PreservePVCByDefault: true,
			},
		},
		&GcCfg{})
	assert.NoError(t, err)
	assert.True(t, b)

	b, err = getPreservePersistentVolumeClaims(
		&config.Config{
			Kubernetes: &config.Kubernetes{
				PreservePVCByDefault: true,
			},
		},
		&GcCfg{
			DiscardPersistentVolumeClaims: true,
		})
	assert.NoError(t, err)
	assert.False(t, b)

	b, err = getPreservePersistentVolumeClaims(
		&config.Config{},
		&GcCfg{
			PreservePersistentVolumeClaims: true,
			DiscardPersistentVolumeClaims:  true,
		})
	assert.Error(t, err)
}

type mockGcClient struct {
	calls []gcCall
}

type gcCall struct {
	name    names.Name
	options *k8s.GcOptions
}

func (gcClient *mockGcClient) Gc(ctx context.Context, cfg *config.Config, name names.Name, options *k8s.GcOptions) error {
	gcClient.calls = append(gcClient.calls, gcCall{
		name,
		options,
	})
	return nil
}
