// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package container

import (
	"bufio"
	"context"
	"errors"
	"github.com/hchauvin/warp/pkg/config"
	"golang.org/x/sync/errgroup"
	"os/exec"
	"regexp"
	"strings"
)

type docker struct {
	path string
}

var (
	classicImageIDRe  = regexp.MustCompile(`^Successfully built ([a-z0-9]+)$`)
	buildkitImageIDRe = regexp.MustCompile(`^#[0-9]+\swriting image\s(sha256:([a-z0-9]+))\s`)
)

func (dk *docker) build(ctx context.Context, cfg *config.Config, fromRef string, labels map[string]string) (nextRef string, err error) {
	args := []string{"build"}
	for k, v := range labels {
		args = append(args, "--label", k+"="+v)
	}
	args = append(args, "-")
	cmd := exec.CommandContext(ctx, dk.path, args...)
	cmd.Stdin = strings.NewReader("FROM " + fromRef)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", err
	}

	var imageID string
	var scannersg errgroup.Group
	scannersg.Go(func() error {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			cfg.Logger().Info("container.build", string(scanner.Bytes()))
			match := classicImageIDRe.FindSubmatch(scanner.Bytes())
			if match != nil {
				imageID = string(match[1])
				break
			}
		}
		return nil
	})
	scannersg.Go(func() error {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			cfg.Logger().Info("container.build", string(scanner.Bytes()))
			match := buildkitImageIDRe.FindSubmatch(scanner.Bytes())
			if match != nil {
				imageID = string(match[1])
				break
			}
		}
		return nil
	})

	if err := cmd.Wait(); err != nil {
		return "", err
	}
	if err := scannersg.Wait(); err != nil {
		return "", err
	}
	if imageID == "" {
		return "", errors.New("could not find (short) image ID in stdout or stderr of 'docker build'")
	}

	if !strings.HasPrefix(imageID, "sha256") {
		out, err := exec.CommandContext(ctx, dk.path, "inspect", "--format", "{{ .Id }}", imageID).Output()
		if err != nil {
			return "", err
		}
		imageID = strings.TrimSpace(string(out))
	}

	return nextRef, nil
}

func (dk *docker) tag(ctx context.Context, cfg *config.Config, ref string, nextRef string) error {
	cmd := exec.CommandContext(ctx, dk.path, "tag", ref, nextRef)
	cfg.Logger().Pipe("container.tag", cmd)
	return cmd.Run()
}

func (dk *docker) push(ctx context.Context, cfg *config.Config, ref string) error {
	cmd := exec.CommandContext(ctx, dk.path, "push", ref)
	cfg.Logger().Pipe("container.push", cmd)
	return cmd.Run()
}
