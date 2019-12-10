package mongo

import (
	"errors"
	"fmt"
	"strings"
)

type options struct {
	uri        string
	database   string
	collection string
}

func parseConnectionString(backendURL string) (*options, error) {
	components := strings.Split(backendURL, ";")

	opts := &options{}
	for _, s := range components {
		components := strings.SplitN(s, "=", 2)
		if len(components) != 2 {
			return nil, errors.New("URL format error: options must have format \"key=value\"")
		}
		key := components[0]
		value := components[1]
		switch key {
		case "uri":
			opts.uri = value

		case "database":
			opts.database = value

		case "collection":
			opts.collection = value

		default:
			return nil, fmt.Errorf("unrecognized option \"%s\"", key)
		}
	}

	if opts.uri == "" {
		return nil, errors.New("uri option is mandatory")
	}
	if opts.database == "" {
		return nil, errors.New("database option is mandatory")
	}
	if opts.collection == "" {
		return nil, errors.New("collection option is mandatory")
	}

	return opts, nil
}
