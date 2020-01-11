// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package batch

import (
	"context"
	"github.com/hchauvin/name_manager/pkg/name_manager"
	"github.com/hchauvin/warp/pkg/log"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
	"strconv"
	"testing"
)

func TestHoldOneNonExclusive(t *testing.T) {
	const pipelineName = "pipeline"

	h := newStackHolder()
	info, err := h.hold(context.Background(), &log.Logger{}, pipelineName, holdConfig{
		maxStacksPerPipeline: 1,
		exclusive:            false,
		hold: func() (*names.Name, <-chan error, name_manager.ReleaseFunc, error) {
			errc := make(chan error)
			release := func() error {
				close(errc)
				return nil
			}
			return &names.Name{ShortName: "foo"}, errc, release, nil
		},
	})
	assert.NoError(t, err)

	assert.Equal(t, "foo", info.name.String())
	assert.Equal(t, pipelineName, info.pipelineName)
	assert.False(t, info.exclusiveLock)
	assert.Equal(t, 1, info.usageCount)

	assert.Equal(t, int64(1), h.stackCount.Load())
	assert.Equal(t, int64(1), h.freeStackCount.Load())

	h.release(info.name.String())

	info = h.stacks[info.name.String()]
	assert.Equal(t, 0, info.usageCount)

	assert.Equal(t, int64(1), h.stackCount.Load())
	assert.Equal(t, int64(1), h.freeStackCount.Load())

	assert.Len(t, h.stacks, 1)
}

func TestHoldTwoNonExclusive(t *testing.T) {
	const pipelineName = "pipeline"

	h := newStackHolder()
	var stackCount atomic.Int64
	holdCfg := holdConfig{
		maxStacksPerPipeline: 1,
		exclusive:            false,
		hold: func() (*names.Name, <-chan error, name_manager.ReleaseFunc, error) {
			errc := make(chan error)
			release := func() error {
				close(errc)
				return nil
			}
			return &names.Name{Family: "foo", ShortName: strconv.FormatInt(stackCount.Inc(), 10)}, errc, release, nil
		},
	}

	info1, err := h.hold(context.Background(), &log.Logger{}, pipelineName, holdCfg)
	assert.NoError(t, err)
	assert.Equal(t, "foo-1", info1.name.DNSName())
	assert.Equal(t, 1, info1.usageCount)

	info2, err := h.hold(context.Background(), &log.Logger{}, pipelineName, holdCfg)
	assert.NoError(t, err)
	assert.Equal(t, "foo-1", info2.name.DNSName())
	assert.Equal(t, 2, info2.usageCount)

	assert.Len(t, h.stacks, 1)

	assert.Equal(t, int64(1), h.stackCount.Load())
	assert.Equal(t, int64(1), h.freeStackCount.Load())

	h.release("foo_1")

	info := h.stacks["foo_1"]
	assert.Equal(t, 1, info.usageCount)

	assert.Equal(t, int64(1), h.stackCount.Load())
	assert.Equal(t, int64(1), h.freeStackCount.Load())

	assert.Len(t, h.stacks, 1)
}

func TestHoldExclusive(t *testing.T) {
	const pipelineName = "pipeline"

	var stackCount atomic.Int64
	hold := func() (*names.Name, <-chan error, name_manager.ReleaseFunc, error) {
		errc := make(chan error)
		release := func() error {
			close(errc)
			return nil
		}
		return &names.Name{Family: "foo", ShortName: strconv.FormatInt(stackCount.Inc(), 10)}, errc, release, nil
	}

	h := newStackHolder()

	info1, err := h.hold(context.Background(), &log.Logger{}, pipelineName, holdConfig{
		maxStacksPerPipeline: 2,
		exclusive:            false,
		hold:                 hold,
	})
	assert.NoError(t, err)
	assert.Equal(t, "foo-1", info1.name.DNSName())
	assert.Equal(t, 1, info1.usageCount)

	info2, err := h.hold(context.Background(), &log.Logger{}, pipelineName, holdConfig{
		maxStacksPerPipeline: 2,
		exclusive:            true,
		hold:                 hold,
	})
	assert.NoError(t, err)
	assert.Equal(t, "foo-2", info2.name.DNSName())
	assert.Equal(t, 1, info2.usageCount)

	info3, err := h.hold(context.Background(), &log.Logger{}, pipelineName, holdConfig{
		maxStacksPerPipeline: 2,
		exclusive:            false,
		hold:                 hold,
	})
	assert.NoError(t, err)
	assert.Equal(t, "foo-1", info3.name.DNSName())
	assert.Equal(t, 2, info3.usageCount)

	assert.Len(t, h.stacks, 2)

	assert.Equal(t, int64(2), h.stackCount.Load())
	assert.Equal(t, int64(1), h.freeStackCount.Load())

	h.release("foo_2")

	assert.Equal(t, int64(2), h.stackCount.Load())
	assert.Equal(t, int64(2), h.freeStackCount.Load())
}

func TestHoldTwoWithWait(t *testing.T) {
	const pipelineName = "pipeline"

	var stackCount atomic.Int64
	hold := func() (*names.Name, <-chan error, name_manager.ReleaseFunc, error) {
		errc := make(chan error)
		release := func() error {
			close(errc)
			return nil
		}
		return &names.Name{Family: "foo", ShortName: strconv.FormatInt(stackCount.Inc(), 10)}, errc, release, nil
	}

	h := newStackHolder()

	info1, err := h.hold(context.Background(), &log.Logger{}, pipelineName, holdConfig{
		maxStacksPerPipeline: 1,
		exclusive:            true,
		hold:                 hold,
	})
	assert.NoError(t, err)
	assert.Equal(t, "foo-1", info1.name.DNSName())
	assert.Equal(t, 1, info1.usageCount)

	waitc := make(chan struct{})
	donec := make(chan struct{})

	go func() {
		info2, err := h.hold(context.Background(), &log.Logger{}, pipelineName, holdConfig{
			maxStacksPerPipeline: 1,
			exclusive:            true,
			hold:                 hold,
			waitc:                waitc,
		})
		assert.NoError(t, err)
		assert.Equal(t, "foo-1", info2.name.DNSName())
		assert.Equal(t, 1, info2.usageCount)
		close(donec)
	}()

	<-waitc
	h.release("foo_1")
	<-donec

	assert.Equal(t, int64(1), h.stackCount.Load())
	assert.Equal(t, int64(0), h.freeStackCount.Load())
}
