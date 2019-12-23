package testing

import (
	"fmt"
	"github.com/avast/retry-go"
	"github.com/stretchr/testify/assert"
	"net"
	"net/http"
	"testing"
)

func TestExpectBody(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	go func() {
		err = retry.Do(func() error { return http.Serve(listener, &handler{}) })
		assert.NoError(t, err)
	}()

	err = ExpectBody(fmt.Sprintf("http://127.0.0.1:%d", port), "__body__")
	assert.NoError(t, err)
}

type handler struct{}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, err := w.Write([]byte("__body__")); err != nil {
		panic(err.Error())
	}
}
