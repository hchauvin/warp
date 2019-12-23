// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package telemetry

import (
	"fmt"
	"regexp"
)

type Client interface {
	Send(payload interface{})
	Close()
}

type Backend struct {
	Protocol  string
	NewClient func(connectionString string) (Client, error)
}

var backends = make(map[string]Backend)

func RegisterBackend(backend Backend) {
	if _, ok := backends[backend.Protocol]; ok {
		panic(fmt.Sprintf("backend '%s' is already registered", backend.Protocol))
	}
	backends[backend.Protocol] = backend
}

func NewClient(url string) (Client, error) {
	backendProtocol, backendConnectionString, err := parseConnectionString(url)
	if err != nil {
		return nil, err
	}

	backend, ok := backends[backendProtocol]
	if !ok {
		return nil, fmt.Errorf("backend '%s' has not been registered", backendProtocol)
	}

	return backend.NewClient(backendConnectionString)
}

var parseURLRe = regexp.MustCompile("^([^:]+)://(.+)$")

func parseConnectionString(url string) (backendProtocol string, backendConnectionString string, err error) {
	submatches := parseURLRe.FindStringSubmatch(url)
	if submatches == nil {
		return "", "", fmt.Errorf("telemetry: invalid backend connection string: '%s'", url)
	}
	if len(submatches) != 3 {
		panic("expected 3 submatches")
	}
	backendProtocol = submatches[1]
	backendConnectionString = submatches[2]
	return
}
