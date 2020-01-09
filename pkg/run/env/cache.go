// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package env

import (
	"fmt"
	"sync"
)

type templateFuncsCache struct {
	cacheMut sync.RWMutex
	cache    map[string]cacheEntry
}

type cacheEntry struct {
	string
	error
}

func newTemplateFuncsCache() templateFuncsCache {
	return templateFuncsCache{
		sync.RWMutex{},
		make(map[string]cacheEntry),
	}
}

func (funcs *templateFuncsCache) memoize(f func() (string, error), fname string, args ...interface{}) (string, error) {
	hash := fmt.Sprintf("%s %v", fname, args)
	funcs.cacheMut.RLock()
	cached, ok := funcs.cache[hash]
	funcs.cacheMut.RUnlock()
	if ok {
		return cached.string, cached.error
	}
	ans, err := f()
	funcs.cacheMut.Lock()
	funcs.cache[hash] = cacheEntry{ans, err}
	funcs.cacheMut.Unlock()
	return ans, err
}
