// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package batches

import (
	"github.com/go-playground/validator"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}
