// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

func toolPath(tool Tool, workspaceDir, path string) (fullPath string, err error) {
	if path == "" {
		panic(fmt.Sprintf("unexpected empty path for tool %s", tool))
	}
	if filepath.IsAbs(path) {
		return path, nil
	}
	if !strings.Contains(path, "/") {
		return path, nil
	}
	if strings.HasPrefix(path, "./") {
		path = path[2:]
	}
	return filepath.Join(workspaceDir, path), nil
}
