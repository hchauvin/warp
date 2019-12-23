// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

// Package mongo implements telemetry on top of MongoDB.
package mongo

import (
	"context"
	"fmt"
	"github.com/hchauvin/warp/pkg/telemetry"
	"go.mongodb.org/mongo-driver/mongo"
	mongo_options "go.mongodb.org/mongo-driver/mongo/options"
	"reflect"
	"time"
)

func init() {
	telemetry.RegisterBackend(telemetry.Backend{
		Protocol:  "mongo",
		NewClient: newClient,
	})
}

const appName = "warp"

type client struct {
	options *options
	client  *mongo.Client
}

func newClient(connectionString string) (telemetry.Client, error) {
	options, err := parseConnectionString(connectionString)
	if err != nil {
		return nil, err
	}

	mongoConnectCtx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	c, err := mongo.Connect(mongoConnectCtx, mongo_options.Client().ApplyURI(options.uri))
	if err != nil {
		return nil, err
	}
	return &client{
		options,
		c,
	}, nil
}

type telemetryDocument struct {
	App     string `bson:"app"`
	Type    string `bson:"type"`
	Payload interface{}
}

// Send implements telemetry.Client.
func (mongo *client) Send(payload interface{}) {
	go func() {
		doc := telemetryDocument{
			App:     appName,
			Type:    getType(payload),
			Payload: payload,
		}
		_, err := mongo.client.
			Database(mongo.options.database).
			Collection(mongo.options.collection).
			InsertOne(context.TODO(), doc)
		if err != nil {
			fmt.Printf("ERROR: Cannot send telemetry event: %v\n", err)
		}
	}()
}

// Close implements telemetry.Client.
func (mongo *client) Close() {
	mongo.client.Disconnect(context.TODO())
}

func getType(myvar interface{}) string {
	t := reflect.TypeOf(myvar)
	if t.Kind() == reflect.Ptr {
		return "*" + t.Elem().Name()
	}
	return t.Name()
}
