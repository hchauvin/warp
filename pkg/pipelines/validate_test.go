package pipelines

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestValidateName(t *testing.T) {
	err := validate.Struct(struct {
		Name string `validate:"name"`
	}{
		Name: "valid",
	})
	assert.NoError(t, err)

	err = validate.Struct(struct {
		Name string `validate:"name"`
	}{
		Name: "$",
	})
	assert.Error(t, err)

	err = validate.Struct(struct {
		Names []string `validate:"name"`
	}{
		Names: []string{"valid"},
	})
	assert.NoError(t, err)

	err = validate.Struct(struct {
		Names []string `validate:"name"`
	}{
		Names: []string{"$"},
	})
	assert.Error(t, err)
}
