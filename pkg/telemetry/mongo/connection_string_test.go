package mongo

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseConnectionString(t *testing.T) {
	o, err := parseConnectionString("uri=foo;database=bar;collection=qux")
	assert.NoError(t, err)
	assert.Equal(t, &options{uri: "foo", database: "bar", collection: "qux"}, o)

	_, err = parseConnectionString("uri=foo;invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "options must have format")

	_, err = parseConnectionString("unknown=foo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unrecognized option")

	_, err = parseConnectionString("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "the following options are mandatory: uri, database, collection")
}
