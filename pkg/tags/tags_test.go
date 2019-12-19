package tags

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNoFilter(t *testing.T) {
	filter, err := CompileFilter("")
	assert.NoError(t, err)
	assert.Len(t, filter.excludeTagSet, 0)
	assert.Len(t, filter.includeTagSet, 0)

	assert.EqualValues(t, true, filter.Apply([]string{"foo", "bar"}))
	assert.EqualValues(t, true, filter.Apply(nil))
}

func TestIncludeFilter(t *testing.T) {
	filter, err := CompileFilter("foo")
	assert.NoError(t, err)
	assert.Len(t, filter.excludeTagSet, 0)
	assert.Len(t, filter.includeTagSet, 1)

	assert.EqualValues(t, true, filter.Apply([]string{"foo"}))
	assert.EqualValues(t, false, filter.Apply([]string{"bar"}))
	assert.EqualValues(t, false, filter.Apply(nil))
	assert.EqualValues(t, true, filter.Apply([]string{"foo", "bar"}))
}

func TestExcludeFilter(t *testing.T) {
	filter, err := CompileFilter("-foo")
	assert.NoError(t, err)
	assert.Len(t, filter.excludeTagSet, 1)
	assert.Len(t, filter.includeTagSet, 0)

	assert.EqualValues(t, false, filter.Apply([]string{"foo"}))
	assert.EqualValues(t, true, filter.Apply([]string{"bar"}))
	assert.EqualValues(t, true, filter.Apply(nil))
	assert.EqualValues(t, false, filter.Apply([]string{"foo", "bar"}))

	filter2, err := CompileFilter("!foo")
	assert.NoError(t, err)
	assert.EqualValues(t, filter, filter2)
}

func TestExcludeTrumpsIncludeFilter(t *testing.T) {
	filter, err := CompileFilter("-foo,bar")
	assert.NoError(t, err)
	assert.Len(t, filter.excludeTagSet, 1)
	assert.Len(t, filter.includeTagSet, 1)

	assert.EqualValues(t, false, filter.Apply([]string{"foo"}))
	assert.EqualValues(t, false, filter.Apply([]string{"foo", "bar"}))
	assert.EqualValues(t, true, filter.Apply([]string{"bar"}))
}

func TestIncludeUnionFilter(t *testing.T) {
	filter, err := CompileFilter("foo,bar")
	assert.NoError(t, err)
	assert.Len(t, filter.excludeTagSet, 0)
	assert.Len(t, filter.includeTagSet, 2)

	assert.EqualValues(t, true, filter.Apply([]string{"foo"}))
	assert.EqualValues(t, true, filter.Apply([]string{"bar"}))
}
