package test_utilities

import (
	"DiscoveryStreams/config"
	"context"
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-redis/redis"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

var tools *config.Tools

func TestSetup() *config.Tools {
	rawJSON := []byte(`{
	  "level": "debug",
	  "encoding": "json",
	  "outputPaths": ["stdout"],
	  "errorOutputPaths": ["stderr"],
	  "encoderConfig": {
	    "messageKey": "message",
	    "levelKey": "level",
	    "levelEncoder": "lowercase"
	  }
	}`)

	var cfg zap.Config
	if err := json.Unmarshal(rawJSON, &cfg); err != nil {
		panic(err)
	}
	logger, _ := cfg.Build()
	defer logger.Sync() // flushes buffer, if any

	cache := redis.NewClient(&redis.Options{
		Addr:         os.Getenv("REDIS_ADDRESS"),
		Password:     os.Getenv("REDIS_PASSWORD"),
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})
	tools = &config.Tools{Cache: cache, Logger: logger}

	return tools
}
func GenerateFakeTestToken() string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp":       time.Now().Add(time.Hour * 1).Unix(),
		"iat":       time.Now().Unix(),
		"email":     "test@example",
		"firstname": "Mister",
		"lastname":  "Test",
		"wholename": "Mister" + " " + "Test",
		"jti":       uuid.New().String(),
	})

	tokenString, _ := token.SignedString([]byte(os.Getenv("TOKEN_SECRET")))

	return tokenString
}
func FlushRedis() error {
	_, err := tools.Cache.FlushDB().Result()
	return err
}
func GetMongoDBClient() (*mongo.Client, error) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_URI")))
	return client, err
}
func TestRequest(t *testing.T, ts *httptest.Server, method, path string, body io.Reader, token string) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, body)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}
	if token != "" {
		req.Header.Set("Authorization", "BEARER "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}
	defer resp.Body.Close()

	return resp, string(respBody)
}
