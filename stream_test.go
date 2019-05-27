package main

import (
	"context"
	"fmt"
	"github.com/go-chi/chi"
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

var sugar *zap.SugaredLogger

func TestMain(m *testing.M) {
	os.Setenv("MONGO_URI", "mongodb://localhost:5002")
	os.Setenv("MONGO_NAME", "reach-engine")
	os.Setenv("ADS_URL", "https://coding-challenge.dsc.tv/v1/ads/")
	logger, _ := zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any
	sugar = logger.Sugar()
	code := m.Run()
	os.Exit(code)
}
func TestStreamController_GetStreamGood(t *testing.T) {
	client, err := getMongoDBClient()
	if err != nil {
		t.Fatal(err)
	}

	streamController := NewStreamController(client, os.Getenv("MONGO_NAME"), sugar)

	r := chi.NewRouter()

	r.Route("/v1/streams", func(v1 chi.Router) {
		v1.Route("/{id}", func(sid chi.Router) {
			sid.Use(streamController.StreamCtx)
			sid.Get("/", streamController.GetStream)
		})
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	if _, body := testRequest(t, ts, "GET", "/v1/streams/5938b99cb6906eb1fbaf1f1c", nil); body != goodCase {
		t.Fatalf(body)
	}
}
func TestStreamController_GetStreamNotFound(t *testing.T) {
	client, err := getMongoDBClient()
	if err != nil {
		t.Fatal(err)
	}

	streamController := NewStreamController(client, os.Getenv("MONGO_NAME"), sugar)

	r := chi.NewRouter()

	r.Route("/v1/streams", func(v1 chi.Router) {
		v1.Route("/{id}", func(sid chi.Router) {
			sid.Use(streamController.StreamCtx)
			sid.Get("/", streamController.GetStream)
		})
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	if resp, _ := testRequest(t, ts, "GET", "/v1/streams/59", nil); resp.StatusCode != 404 {
		t.Fatalf(resp.Status)
	}

	if resp, _ := testRequest(t, ts, "GET", "/v1/streams/", nil); resp.StatusCode != 404 {
		t.Fatalf(resp.Status)
	}
}
func TestStreamController_GetStreamAdsServiceDown(t *testing.T) {
	os.Setenv("ADS_URL", "http://localhost:9000/")
	client, err := getMongoDBClient()
	if err != nil {
		t.Fatal(err)
	}

	streamController := NewStreamController(client, os.Getenv("MONGO_NAME"), sugar)

	r := chi.NewRouter()

	r.Route("/v1/streams", func(v1 chi.Router) {
		v1.Route("/{id}", func(sid chi.Router) {
			sid.Use(streamController.StreamCtx)
			sid.Get("/", streamController.GetStream)
		})
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	if resp, _ := testRequest(t, ts, "GET", "/v1/streams/5938b99cb6906eb1fbaf1f1c", nil); resp.StatusCode != 503 {
		t.Fatalf(fmt.Sprintf("%d", resp.StatusCode))
	}
}
func TestStreamController_GetStreamDatabaseDown(t *testing.T) {
	client, err := getMongoDBClient()
	if err != nil {
		t.Fatal(err)
	}

	streamController := NewStreamController(client, os.Getenv("MONGO_NAME"), sugar)
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	client.Disconnect(ctx)

	r := chi.NewRouter()

	r.Route("/v1/streams", func(v1 chi.Router) {
		v1.Route("/{id}", func(sid chi.Router) {
			sid.Use(streamController.StreamCtx)
			sid.Get("/", streamController.GetStream)
		})
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	if resp, _ := testRequest(t, ts, "GET", "/v1/streams/5938b99cb6906eb1fbaf1f1c", nil); resp.StatusCode != 503 {
		t.Fatalf(fmt.Sprintf("%d", resp.StatusCode))
	}
}
func TestStreamController_GetStreamDatabaseNameMissing(t *testing.T) {
	client, err := getMongoDBClient()
	if err != nil {
		t.Fatal(err)
	}

	streamController := NewStreamController(client, "", sugar)

	r := chi.NewRouter()

	r.Route("/v1/streams", func(v1 chi.Router) {
		v1.Route("/{id}", func(sid chi.Router) {
			sid.Use(streamController.StreamCtx)
			sid.Get("/", streamController.GetStream)
		})
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	if resp, _ := testRequest(t, ts, "GET", "/v1/streams/5938b99cb6906eb1fbaf1f1c", nil); resp.StatusCode != 503 {
		t.Fatalf(fmt.Sprintf("%d", resp.StatusCode))
	}
}
func getMongoDBClient() (*mongo.Client, error) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_URI")))
	return client, err
}
func testRequest(t *testing.T, ts *httptest.Server, method, path string, body io.Reader) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, body)
	if err != nil {
		t.Fatal(err)
		return nil, ""
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

const goodCase = `{
  "id": "5938b99cb6906eb1fbaf1f1c",
  "streamUrl": "https://devstreaming-cdn.apple.com/videos/streaming/examples/bipbop_4x3/bipbop_4x3_variant.m3u8",
  "captions": {
    "vtt": {
      "en": "https:\/\/captionslocation.com\/0123456789\/captions.vtt"
    },
    "scc": {
      "en": "https:\/\/captionslocation.com\/0123456789\/captions.scc"
    }
  },
  "ads": {
    "breakOffsets": [
      {
        "index": 0,
        "timeOffset": 0
      },
      {
        "index": 1,
        "timeOffset": 553.07
      },
      {
        "index": 2,
        "timeOffset": 1212.0304583333
      }
    ],
    "breaks": [
      {
        "ads": [],
        "breakId": "0.0.0.125406899",
        "duration": 0,
        "events": {
          "impressions": [
            "http:\/\/some-ad-server.com\/ad\/l\/1"
          ]
        },
        "position": "preroll",
        "timeOffset": 0,
        "type": "linear"
      },
      {
        "ads": [
          {
            "creative": "abcd1234",
            "duration": 30,
            "events": {
              "impressions": [
                "http:\/\/some-ad-server.com\/ad\/l\/1"
              ]
            }
          },
          {
            "creative": "abcd1235",
            "duration": 40,
            "events": {
              "impressions": [
                "http:\/\/some-ad-server.com\/ad\/l\/1"
              ]
            }
          },
          {
            "creative": "abcd1236",
            "duration": 20,
            "events": {
              "impressions": [
                "http:\/\/some-ad-server.com\/ad\/l\/1"
              ]
            }
          }
        ],
        "breakId": "mid0",
        "duration": 90,
        "events": {
          "impressions": [
            "http:\/\/some-ad-server.com\/ad\/l\/1"
          ]
        },
        "position": "midroll",
        "timeOffset": 553.07,
        "type": "linear"
      },
      {
        "ads": [
          {
            "creative": "abcd1237",
            "duration": 10,
            "events": {
              "impressions": [
                "http:\/\/some-ad-server.com\/ad\/l\/1"
              ]
            }
          },
          {
            "creative": "abcd1238",
            "duration": 20,
            "events": {
              "impressions": [
                "http:\/\/some-ad-server.com\/ad\/l\/1"
              ]
            }
          },
          {
            "creative": "abcd1239",
            "duration": 30,
            "events": {
              "impressions": [
                "http:\/\/some-ad-server.com\/ad\/l\/1"
              ]
            }
          }
        ],
        "breakId": "mid1",
        "duration": 60,
        "events": {
          "impressions": [
            "http:\/\/some-ad-server.com\/ad\/l\/1"
          ]
        },
        "position": "midroll",
        "timeOffset": 1212.0304583333,
        "type": "linear"
      }
    ]
  }
}`
