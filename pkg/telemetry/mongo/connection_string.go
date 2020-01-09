// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

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
	var components []string
	if backendURL != "" {
		components = strings.Split(backendURL, ";")
	}

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

	var missingOptions []string
	if opts.uri == "" {
		missingOptions = append(missingOptions, "uri")
	}
	if opts.database == "" {
		missingOptions = append(missingOptions, "database")
	}
	if opts.collection == "" {
		missingOptions = append(missingOptions, "collection")
	}

	if len(missingOptions) > 0 {
		return nil, fmt.Errorf(
			"the following options are mandatory: %s",
			strings.Join(missingOptions, ", "))
	}

	return opts, nil
}
