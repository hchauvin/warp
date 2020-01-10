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

func TestHoldNamedStack(t *testing.T) {
	p := &pipelines.Pipeline{
		Stack: pipelines.Stack{
			Name: "foo",
		},
	}

	name, errc, release, err := Hold(&config.Config{}, p)
	assert.NoError(t, err)
	defer release()
	go func() {
		err := <-errc
		assert.NoError(t, err)
	}()

	assert.Equal(t, "foo", name.String())
}

func TestHoldStack(t *testing.T) {
	nmDB, err := ioutil.TempFile("", "name_manager")
	assert.NoError(t, err)
	defer os.Remove(nmDB.Name())

	cfg := &config.Config{
		NameManagerURL: "local://" + nmDB.Name(),
	}
	p := &pipelines.Pipeline{
		Stack: pipelines.Stack{
			Family: "foo",
		},
	}

	name, errc, release, err := Hold(cfg, p)
	assert.NoError(t, err)
	go func() {
		err := <-errc
		assert.NoError(t, err)
	}()

	assert.Equal(t, "foo", name.Family)
	assert.Equal(t, "0", name.ShortName)

	nm, err := name_manager.CreateFromURL(cfg.NameManagerURL)
	assert.NoError(t, err)

	names, err := nm.List()
	assert.NoError(t, err)

	assert.Len(t, names, 1)
	assert.Equal(t, "foo", names[0].Family)
	assert.Equal(t, "0", names[0].Name)
	assert.False(t, names[0].Free)

	release()

	names, err = nm.List()
	assert.NoError(t, err)

	assert.Len(t, names, 1)
	assert.True(t, names[0].Free)
}

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
