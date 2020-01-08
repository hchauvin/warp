// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package interactive

import (
	"github.com/benbjohnson/clock"
	"github.com/hchauvin/warp/pkg/log"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestInteractive(t *testing.T) {
	logger := &log.Logger{}
	eventc := make(chan interface{})
	done := make(chan struct{})
	clk := clock.NewMock()
	clk.Set(initialTestTime)
	tickerc := make(chan time.Time)
	r := newMockReporter()

	go func() {
		err := report(logger, eventc, done, clk, tickerc, r)
		assert.NoError(t, err)
	}()

	tick := func() {
		clk.Add(refreshDuration)
		tickerc <- time.Now()
	}

	tick()
	assert.Equal(t, []string{"Running [0/0, 0.0s]:"}, r.lastCall())

	eventc <- SetStateEvent{Name: "foo", State: Initial, Stage: "stage"}
	tick()
	assert.Equal(t, []string{"Running [0/1, 0.1s]:"}, r.lastCall())

	eventc <- SetStateEvent{Name: "foo", State: Started, Stage: "stage"}
	tick()
	assert.Equal(t, []string{"Running [0/1, 0.1s]:", "=> [ 0.0s] foo stage"}, r.lastCall())

	tick()
	assert.Equal(t, []string{"Running [0/1, 0.1s]:", "=> [ 0.1s] foo stage"}, r.lastCall())

	eventc <- SetStateEvent{Name: "bar", State: Initial, Stage: "stage"}
	tick()
	assert.Equal(t, []string{"Running [0/2, 0.2s]:", "=> [ 0.1s] foo stage"}, r.lastCall())

	eventc <- SetStateEvent{Name: "bar", State: Started, Stage: "stage2"}
	tick()
	assert.Equal(
		t,
		[]string{
			"Running [0/2, 0.2s]:",
			"=> [ 0.0s] bar stage2",
			"=> [ 0.1s] foo stage",
		},
		r.lastCall())

	eventc <- SetStateEvent{Name: "foo", State: Completed}
	tick()
	assert.Equal(
		t,
		[]string{
			"Running [1/2, 0.3s]:",
			"=> [ 0.1s] bar stage2",
			"=> [ 0.2s] foo completed",
		},
		r.lastCall())

	clk.Add(allocationPersistenceDuration)
	tick()
	assert.Equal(
		t,
		[]string{
			"Running [1/2, 0.8s]:",
			"=> [ 0.6s] bar stage2",
		},
		r.lastCall())

	close(done)

	assert.Equal(t, []string(nil), r.lastCall())
	assert.Equal(t, summary{totalDuration: 850 * time.Millisecond}, r.summary())
}

type mockReporter struct {
	callc    chan []string
	summaryc chan summary
}

func newMockReporter() *mockReporter {
	return &mockReporter{
		callc:    make(chan []string),
		summaryc: make(chan summary),
	}
}

func (r *mockReporter) replace(lines []string) error {
	r.callc <- lines
	return nil
}

func (r *mockReporter) summarize(s summary) error {
	r.summaryc <- s
	return nil
}

func (r *mockReporter) lastCall() []string {
	return <-r.callc
}

func (r *mockReporter) summary() summary {
	return <-r.summaryc
}

var initialTestTime = time.Unix(0, 0)
