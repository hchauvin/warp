package config

import (
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRead(t *testing.T) {
	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, "/workspace/config.yml", simpleConfigBytes, 0666)
	assert.NoError(t, err)

	cfg, err := ReadFs(fs, "/workspace/config.yml")
	assert.NoError(t, err)

	assert.EqualValues(t, simpleConfig, cfg)
}

var simpleConfigBytes = []byte(`
NameManagerURL = "mongo://xxx"
OutputRoot = ".warp"
`)

var simpleConfig = &Config{
	NameManagerURL: "mongo://xxx",
	OutputRoot:     ".warp",
}
