// Run with `docker run --name some-mongo -p 27017:27017 mongo:bionic`
// export MONGODB_URI="mongodb://127.0.0.1:27017"
// export MONGODB_DATABASE="..." (optional)
package mongo

import (
	"context"
	"errors"
	"fmt"
	"github.com/avast/retry-go"
	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hchauvin/warp/pkg/telemetry"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"os"
	"testing"
	"time"
)

func TestSend(t *testing.T) {
	godotenv.Load("../../../.env")

	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		t.Skip("No MongoDB")
	}

	database := os.Getenv("MONGODB_DATABASE")
	if database == "" {
		database = "telemetry_test"
	}

	collection := petname.Generate(3, "-")

	c, err := telemetry.NewClient(fmt.Sprintf(
		"mongo://uri=%s;database=%s;collection=%s",
		uri, database, collection))
	assert.NoError(t, err)
	defer func() {
		mongoClient := c.(*client).client
		mongoClient.Database(database).
			Collection(collection).
			Drop(context.Background())
		c.Close()
	}()

	versionDate := time.Now().String()
	started := time.Now().UTC()
	c.Send(telemetry.CLIInvocation{
		CLIVersion: telemetry.CLIVersion{
			Version: "__version__",
			Commit:  "__commit__",
			Date:    versionDate,
		},
		User:    "__user__",
		Started: started,
		Args:    []string{"arg0", "arg1"},
	})

	err = retry.Do(func() error {
		ctx := context.Background()
		mongoClient := c.(*client).client
		result, err := mongoClient.
			Database(database).
			Collection(collection).
			Find(ctx, bson.M{})
		if err != nil {
			return err
		}
		if !result.Next(ctx) {
			return errors.New("expected at least one result")
		}
		if result.Err() != nil {
			return result.Err()
		}
		assert.Equal(t, "warp", result.Current.Lookup("app").StringValue())
		assert.Equal(t, "CLIInvocation", result.Current.Lookup("type").StringValue())
		payload := result.Current.Lookup("payload").Document()
		assert.Equal(t, "__version__", payload.Lookup("version").StringValue())
		assert.Equal(t, "__commit__", payload.Lookup("commit").StringValue())
		assert.Equal(t, versionDate, payload.Lookup("date").StringValue())
		assert.Equal(t, "__user__", payload.Lookup("user").StringValue())
		assert.Equal(t, started.Round(time.Microsecond*1000), payload.Lookup("started").Time().UTC())
		args := payload.Lookup("args").Array()
		assert.Equal(t, "arg0", args.Index(0).Value().StringValue())
		assert.Equal(t, "arg1", args.Index(1).Value().StringValue())
		_, err = args.IndexErr(2)
		assert.Equal(t, err, bsoncore.ErrOutOfBounds)
		if result.Next(ctx) {
			return errors.New("expect one and only one result")
		}
		return nil
	})
	assert.NoError(t, err)
}
