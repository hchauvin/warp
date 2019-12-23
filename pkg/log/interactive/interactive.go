// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package interactive

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/hchauvin/warp/pkg/log"
	"sort"
	"sync"
	"time"
)

type allocationProgress struct {
	name      string
	state     State
	stage     string
	started   *time.Time
	completed *time.Time
}

func Report(l *log.Logger, eventc <-chan interface{}, done <-chan struct{}) error {
	start := time.Now()

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
					now := time.Now()
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

	var fixed FixedTerminalLines
	started := time.Now()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			if err := fixed.Replace(nil); err != nil {
				return err
			}
			goto summary
		case <-ticker.C:
		}

		now := time.Now()

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
				if time.Since(*p.completed).Seconds() < 0.5 {
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
		if err := fixed.Replace(lines); err != nil {
			return err
		}
	}

summary:
	fmt.Printf("----------------------------\n")
	fmt.Printf("Total duration: %s\n", time.Now().Sub(start).Round(time.Millisecond))
	return nil
}
