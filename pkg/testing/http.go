// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package testing

import (
	"errors"
	"fmt"
	"github.com/avast/retry-go"
	"io/ioutil"
	"net/http"
	"time"
)

// ExpectBody sends an HTTP GET request to the given request and checks the text body.
func ExpectBody(endpoint string, expectedBody string) error {
	if endpoint == "" {
		return errors.New("endpoint is missing")
	}

	return retry.Do(func() error {
		return expectBody(endpoint, expectedBody)
	}, retry.Attempts(5), retry.Delay(3*time.Second))
}

func expectBody(endpoint, expectedBody string) error {
	resp, err := http.Get(endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if string(b) != expectedBody {
		return fmt.Errorf("unexpected body: expected '%s', got '%s'", expectedBody, b)
	}
	return nil
}
