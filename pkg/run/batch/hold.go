// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package batch

import (
	"context"
	"github.com/hchauvin/name_manager/pkg/name_manager"
	"github.com/hchauvin/warp/pkg/log"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"go.uber.org/atomic"
	"sync"
)

type stackHolder struct {
	stackCount     atomic.Int64
	freeStackCount atomic.Int64
	releasec       chan struct{}
	stacks         map[string]*stackInfo
	stacksMut      sync.Mutex
}

type stackInfo struct {
	pipelineName  string
	name          names.Name
	holdErrc      <-chan error
	release       name_manager.ReleaseFunc
	exclusiveLock bool
	usageCount    int
	deployed      atomic.Bool
	before        atomic.Bool
	deployedc     chan struct{}
	initializedc  chan struct{}
}

func newStackHolder() stackHolder {
	return stackHolder{
		releasec: make(chan struct{}),
		stacks:   make(map[string]*stackInfo),
	}
}

type holdConfig struct {
	maxStacksPerPipeline int
	exclusive            bool
	hold                 func() (*names.Name, <-chan error, name_manager.ReleaseFunc, error)
	waitc                chan struct{}
}

func (holder *stackHolder) hold(ctx context.Context, logger *log.Logger, pipelineName string, cfg holdConfig) (*stackInfo, error) {
	var wait bool
	if cfg.exclusive {
		wait = holder.stackCount.Load() >= int64(cfg.maxStacksPerPipeline)
	} else {
		wait = holder.freeStackCount.Load() == 0 &&
			holder.stackCount.Load() >= int64(cfg.maxStacksPerPipeline)
	}
	if wait {
		logger.Info(
			logDomain,
			"max stacks per pipeline %d reached; waiting for a stack release",
			cfg.maxStacksPerPipeline)
		if cfg.waitc != nil {
			close(cfg.waitc)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-holder.releasec:
		}
	}

	holder.stacksMut.Lock()
	defer holder.stacksMut.Unlock()

	var stack *stackInfo
	if cfg.exclusive {
		for _, st := range holder.stacks {
			if st.usageCount == 0 {
				stack = st
				break
			}
		}
		if stack != nil {
			stack.usageCount = 1
			stack.exclusiveLock = true
			holder.freeStackCount.Dec()
			return stack, nil
		}
	} else {
		for _, st := range holder.stacks {
			if !st.exclusiveLock {
				stack = st
				break
			}
		}
		if stack != nil {
			stack.usageCount++
			return stack, nil
		}
	}

	name, holdErrc, release, err := cfg.hold()
	if err != nil {
		return nil, err
	}

	stack = &stackInfo{
		pipelineName:  pipelineName,
		name:          *name,
		release:       release,
		holdErrc:      holdErrc,
		usageCount:    1,
		exclusiveLock: cfg.exclusive,
		deployedc:     make(chan struct{}),
		initializedc:  make(chan struct{}),
	}
	holder.stacks[name.String()] = stack
	holder.stackCount.Inc()
	if !cfg.exclusive {
		holder.freeStackCount.Inc()
	}
	return stack, nil
}

func (holder *stackHolder) release(stackName string) {
	holder.stacksMut.Lock()
	defer holder.stacksMut.Unlock()

	stack := holder.stacks[stackName]
	stack.usageCount--
	if stack.exclusiveLock {
		stack.before.Store(false)
		holder.freeStackCount.Inc()
	}
	stack.exclusiveLock = false
	go func() {
		holder.releasec <- struct{}{}
	}()
}
