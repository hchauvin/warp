package run

import (
	"bytes"
	"context"
	"fmt"
	"github.com/hchauvin/warp/pkg/log"
	"github.com/hchauvin/warp/pkg/pipelines"
	"github.com/hchauvin/warp/pkg/run/env"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestHttpGetSuccess(t *testing.T) {
	s := newHttpGetTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	defer s.Close()

	logWriter := &bytes.Buffer{}

	err := httpGet(
		context.Background(),
		&log.Logger{Writer: logWriter},
		&pipelines.HTTPGet{URL: s.url()},
		env.NewTransformer(),
		time.After,
	)
	assert.NoError(t, err)
}

func TestHttpGetError(t *testing.T) {
	s := newHttpGetTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	})
	defer s.Close()

	logWriter := &bytes.Buffer{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	after := func(d time.Duration) <-chan time.Time {
		c := make(chan time.Time, 1)
		c <- time.Now()
		cancel()
		return c
	}

	err := httpGet(
		ctx,
		&log.Logger{Writer: logWriter},
		&pipelines.HTTPGet{URL: s.url()},
		env.NewTransformer(),
		after,
	)

	assert.Error(t, err)
	assert.Contains(t, logWriter.String(), "non-2xx status code")
	assert.Contains(t, logWriter.String(), "500")
}

func TestHttpGetRetry(t *testing.T) {
	s := newHttpGetTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	})
	defer s.Close()

	logWriter := &bytes.Buffer{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const maxBackoffs = 3
	backoffs := 0
	after := func(d time.Duration) <-chan time.Time {
		c := make(chan time.Time, 1)
		backoffs++
		if backoffs == maxBackoffs {
			cancel()
		} else {
			c <- time.Now()
		}
		return c
	}

	err := httpGet(
		ctx,
		&log.Logger{Writer: logWriter},
		&pipelines.HTTPGet{URL: s.url()},
		env.NewTransformer(),
		after,
	)

	assert.Error(t, err)
	fmt.Printf("LOG %s\n", logWriter.String())
	assert.Equal(t, maxBackoffs, strings.Count(logWriter.String(), "non-2xx status code"))
}

func TestHttpGetHeaders(t *testing.T) {
	s := newHttpGetTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "bar", r.Header.Get("X-Foo"))
		w.WriteHeader(200)
	})
	defer s.Close()

	logWriter := &bytes.Buffer{}

	err := httpGet(
		context.Background(),
		&log.Logger{Writer: logWriter},
		&pipelines.HTTPGet{
			URL:         s.url(),
			HTTPHeaders: []pipelines.HTTPHeader{{"X-Foo", "bar"}},
		},
		env.NewTransformer(),
		time.After,
	)
	assert.NoError(t, err)
}

type httpGetTestServer struct {
	listener net.Listener
	server   *http.Server
}

func newHttpGetTestServer(t *testing.T, handle func(w http.ResponseWriter, r *http.Request)) *httpGetTestServer {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("%v", err)
	}

	router := httprouter.New()
	router.GET("/", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		handle(w, r)
	})
	srv := &http.Server{Handler: router}

	go func() {
		if err := srv.Serve(listener); err != nil {
			if err != http.ErrServerClosed {
				t.Fatalf("%v", err)
			}
		}
	}()

	return &httpGetTestServer{listener, srv}
}

func (s *httpGetTestServer) Close() error {
	if err := s.server.Close(); err != nil {
		return err
	}
	return s.listener.Close()
}

func (s *httpGetTestServer) url() string {
	port := s.listener.Addr().(*net.TCPAddr).Port
	return fmt.Sprintf("http://localhost:%d", port)
}
