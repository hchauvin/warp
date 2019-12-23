// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package interactive

type SetStateEvent struct {
	Name  string
	State State
	Stage string
}

type State string

const (
	Initial   State = "initial"
	Started         = "started"
	Completed       = "completed"
)
