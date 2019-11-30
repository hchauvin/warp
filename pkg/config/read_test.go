package config

import (
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRead(t *testing.T) {
	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, "/workspace/config.yml", simpleConfigBytes, 0666)
	assert.NoError(t, err)

	cfg, err := ReadFs(fs, "/workspace/config.yml")
	assert.NoError(t, err)

	if runtime.GOOS == "windows" {
		assert.Equal(t, filepath.FromSlash("C:/workspace"), cfg.WorkspaceDir)
	} else {
		assert.Equal(t, filepath.FromSlash("/workspace"), cfg.WorkspaceDir)
	}
	assert.Equal(t, "mongo://xxx", cfg.NameManagerURL)
	assert.Equal(t, ".warp", cfg.OutputRoot)
}

var simpleConfigBytes = []byte(`
NameManagerURL = "mongo://xxx"
OutputRoot = ".warp"
`)
