// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package e2e

import (
	"github.com/hchauvin/warp/pkg/warp"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestDev(t *testing.T) {
	godotenv.Load("../.env")

	err := os.MkdirAll("../examples/.warp", 0777)
	if err != nil {
		t.Fatalf("cannot create .wrap output dir: %v", err)
	}

	err = os.MkdirAll("../examples/dev/www", 0777)
	if err != nil {
		t.Fatalf("cannot create www dir: %v", err)
	}

	if err := ioutil.WriteFile("../examples/dev/www/index.html", indexHtml, 0777); err != nil {
		t.Fatalf("cannot write index.html file: %v", err)
	}

	err = warp.Hold(&warp.HoldConfig{
		WorkingDir:   "../examples",
		ConfigPath:   ".warprc.toml",
		PipelinePath: "dev",
		Tail:         true,
		Run:          []string{"test"},
		Rm:           false,
		Dev:          true,
	})
	assert.NoError(t, err)
}

var indexHtml = []byte(`
<html>
<head><title>Hello,</title></head>
<body><h1>World</h1></body>
</html>
`)
