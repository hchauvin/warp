// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package run

import (
	"context"
	"fmt"
	"github.com/hchauvin/warp/pkg/log"
	"github.com/hchauvin/warp/pkg/pipelines"
	"github.com/hchauvin/warp/pkg/run/env"
	"k8s.io/apimachinery/pkg/util/wait"
	"net/http"
	"time"
)

func httpGet(
	ctx context.Context,
	logger *log.Logger,
	spec *pipelines.HTTPGet,
	trans *env.Transformer,
	after func(d time.Duration) <-chan time.Time,
) error {
	url, err := trans.Get(ctx, spec.URL)
	if err != nil {
		return fmt.Errorf("cannot transform env vars: %v", err)
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	for _, h := range spec.HTTPHeaders {
		value, err := trans.Get(ctx, h.Value)
		if err != nil {
			return fmt.Errorf("cannot transform header %s: %v", h.Name, err)
		}
		req.Header.Add(h.Name, value)
	}

	var urlWithResolution string
	if url == spec.URL {
		urlWithResolution = spec.URL
	} else {
		urlWithResolution = fmt.Sprintf("%s (-> %s)", spec.URL, url)
	}

	backoff := wait.Backoff{
		Duration: 200 * time.Millisecond,
		Factor:   3,
		Jitter:   0.1,
		Steps:    10,
		Cap:      4 * time.Second,
	}
	var resp *http.Response
	for {
		resp, err = client.Do(req)
		if err == nil {
			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				err = fmt.Errorf("non-2xx status code: %d %s", resp.StatusCode, resp.Status)
			} else {
				return nil
			}
		}

		logger.Info("run:hook:httpGet", "%s - %v", urlWithResolution, err)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-after(backoff.Step()):
		}
	}
}
