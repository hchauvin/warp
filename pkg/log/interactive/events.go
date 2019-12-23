// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package interactive

// SetStateEvent reports the setting of the state of an
// allocation.
type SetStateEvent struct {
	// Name is the name of the allocation.
	Name string
	// State is the next state of the allocation.
	State State
	// Stage further qualifies the state.
	Stage string
}

// State is the state of an allocation.
type State string

const (
	// Initial is the state an allocation is initially in.
	Initial State = "initial"
	// Started is the state an allocation is in after
	// work has started.
	Started = "started"
	// Completed is the state an allocation is in after
	// work is complete.
	Completed = "completed"
)
