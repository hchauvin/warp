// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package interactive

import (
	"fmt"
	"github.com/benbjohnson/clock"
	"github.com/fatih/color"
	"github.com/hchauvin/warp/pkg/log"
	"sort"
	"sync"
	"time"
)

// Report reports a stream of events in an interactive way, using
// an array of fixed terminal lines.
func Report(l *log.Logger, eventc <-chan interface{}, done <-chan struct{}) error {
	ticker := time.NewTicker(refreshDuration)
	defer ticker.Stop()
	return report(l, eventc, done, clock.New(), ticker.C, &terminalReporter{})
}

// refreshDuration is the duration between two refreshes of the interactive output.
const refreshDuration = 50 * time.Millisecond

// allocationPersistenceDurationSeconds is the number of seconds to persist an
// allocation in the interactive output after it completed.
const allocationPersistenceDuration = 500 * time.Millisecond

type allocationProgress struct {
	name      string
	state     State
	stage     string
	started   *time.Time
	completed *time.Time
}

type summary struct {
	totalDuration time.Duration
}

type reporter interface {
	replace(lines []string) error
	summarize(s summary) error
}

func report(
	l *log.Logger,
	eventc <-chan interface{},
	done <-chan struct{},
	clk clock.Clock,
	tickerc <-chan time.Time,
	r reporter,
) error {
	start := clk.Now()

	l.SetInteractive(true)
	defer l.SetInteractive(false)

	var progressMut sync.Mutex
	progress := make(map[string]allocationProgress)

	go func() {
		for {
			select {
			case <-done:
				return
			case e := <-eventc:
				switch et := e.(type) {
				case SetStateEvent:
					progressMut.Lock()
					p, _ := progress[et.Name]
					now := clk.Now()
					switch et.State {
					case Started:
						p.started = &now
					case Completed:
						p.completed = &now
					}
					p.state = et.State
					p.stage = et.Stage
					progress[et.Name] = p
					progressMut.Unlock()
				}
			}
		}
	}()

	started := clk.Now()
	for {
		select {
		case <-done:
			if err := r.replace(nil); err != nil {
				return err
			}
			goto summary
		case <-tickerc:
		}

		now := clk.Now()

		progressMut.Lock()
		sortedAllocationNames := make([]string, 0, len(progress))
		for allocationName := range progress {
			sortedAllocationNames = append(sortedAllocationNames, allocationName)
		}
		sort.Slice(sortedAllocationNames, func(i, j int) bool {
			return sortedAllocationNames[i] < sortedAllocationNames[j]
		})

		lines := make([]string, 0, len(progress))
		completedCount := 0
		for _, target := range sortedAllocationNames {
			p := progress[target]

			if p.state == Completed {
				completedCount++
			}

			var line string
			bold := color.New(color.Bold).SprintFunc()
			if p.completed != nil {
				if clk.Since(*p.completed) < allocationPersistenceDuration {
					duration := p.completed.Sub(*p.started).Seconds()
					line = fmt.Sprintf(bold("=> [%4.1fs]")+" %s "+bold("%s"), duration, target, p.state)
				}
			} else if p.started != nil {
				duration := now.Sub(*p.started).Seconds()
				line = fmt.Sprintf(bold("=> [%4.1fs]")+" %s %s", duration, target, p.stage)
			}

			if line == "" {
				continue
			}
			lines = append(lines, line)
		}
		summary := fmt.Sprintf(
			"Running [%d/%d, %3.1fs]:",
			completedCount,
			len(progress),
			now.Sub(started).Seconds())
		progressMut.Unlock()
		lines = append([]string{summary}, lines...)
		if err := r.replace(lines); err != nil {
			return err
		}
	}

summary:
	return r.summarize(summary{
		totalDuration: clk.Now().Sub(start).Round(time.Millisecond),
	})
}
