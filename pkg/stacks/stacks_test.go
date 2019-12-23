// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package stacks

import (
	"context"
	"github.com/hchauvin/name_manager/pkg/name_manager"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/pipelines"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestListStacksWithFixedNamePipeline(t *testing.T) {
	cfg := &config.Config{WorkspaceDir: "/workspace"}
	pipeline := &pipelines.Pipeline{
		Stack: pipelines.Stack{
			Name: "foo",
		},
	}

	names, err := List(context.Background(), cfg, pipeline, false)
	assert.NoError(t, err)
	assert.EqualValues(t, []string{"foo"}, names)
}

func TestListStacksWithFamily(t *testing.T) {
	dbf, err := ioutil.TempFile(os.TempDir(), ".db")
	if err != nil {
		t.Fatal(err)
	}
	defer dbf.Close()
	nmURL := "local://" + dbf.Name()

	nm, err := name_manager.CreateFromURL(nmURL)
	defer nm.Reset()
	if err != nil {
		t.Fatal(err)
	}

	name, err := nm.Acquire("foo")
	assert.NoError(t, err)
	assert.Equal(t, "0", name)

	name, err = nm.Acquire("foo")
	assert.NoError(t, err)
	assert.Equal(t, "1", name)

	err = nm.Release("foo", "1")
	assert.NoError(t, err)

	cfg := &config.Config{
		NameManagerURL: nmURL,
		WorkspaceDir:   "/workspace",
	}
	pipeline := &pipelines.Pipeline{
		Stack: pipelines.Stack{
			Family: "foo",
		},
	}

	names, err := List(context.Background(), cfg, pipeline, false)
	assert.NoError(t, err)
	assert.EqualValues(t, []string{"0", "1"}, names)

	names, err = List(context.Background(), cfg, pipeline, true)
	assert.NoError(t, err)
	assert.EqualValues(t, []string{"1"}, names)
}
