package main

import (
	"DiscoveryStreams/api"
	"DiscoveryStreams/test_utilities"
	"context"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/jwtauth"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

var client, _ = test_utilities.GetMongoDBClient()

func TestStreamController_GetStream_Good(t *testing.T) {
	tools := test_utilities.TestSetup()
	streamController := api.NewStreamController(client.Database(os.Getenv("MONGO_DB_NAME")), tools)

	testToken := test_utilities.GenerateFakeTestToken()
	chiRouter := chi.NewRouter()

	chiRouter.Group(func(guarded chi.Router) {
		guarded.Use(VerifyJWT(tools, jwtauth.New("HS256", []byte(os.Getenv("TOKEN_SECRET")), nil)))
		guarded.Route("/v1", func(v1 chi.Router) {
			v1.Route("/streams", func(s chi.Router) {
				s.Route("/{id}", func(sid chi.Router) {
					sid.Get("/", streamController.GetStream)
				})
			})
		})
	})

	ts := httptest.NewServer(chiRouter)
	defer ts.Close()

	if _, body := test_utilities.TestRequest(t, ts, "GET", "/v1/streams/5938b99cb6906eb1fbaf1f1c", nil, testToken); body != goodCase {
		t.Fatalf(body)
	}
}
func TestStreamController_GetStream_NotFound(t *testing.T) {
	tools := test_utilities.TestSetup()
	streamController := api.NewStreamController(client.Database(os.Getenv("MONGO_DB_NAME")), tools)

	testToken := test_utilities.GenerateFakeTestToken()
	chiRouter := chi.NewRouter()

	chiRouter.Group(func(guarded chi.Router) {
		guarded.Use(VerifyJWT(tools, jwtauth.New("HS256", []byte(os.Getenv("TOKEN_SECRET")), nil)))
		guarded.Route("/v1", func(v1 chi.Router) {
			v1.Route("/streams", func(s chi.Router) {
				s.Route("/{id}", func(sid chi.Router) {
					sid.Get("/", streamController.GetStream)
				})
			})
		})
	})
	ts := httptest.NewServer(chiRouter)
	defer ts.Close()

	if resp, _ := test_utilities.TestRequest(t, ts, "GET", "/v1/streams/59", nil, testToken); resp.StatusCode != 404 {
		t.Fatalf(resp.Status)
	}

	if resp, _ := test_utilities.TestRequest(t, ts, "GET", "/v1/streams/", nil, testToken); resp.StatusCode != 404 {
		t.Fatalf(resp.Status)
	}
}
func TestStreamController_GetStream_AdsServiceDown(t *testing.T) {
	os.Setenv("ADS_URL", "http://localhost:9000/")
	tools := test_utilities.TestSetup()
	streamController := api.NewStreamController(client.Database(os.Getenv("MONGO_DB_NAME")), tools)

	testToken := test_utilities.GenerateFakeTestToken()
	chiRouter := chi.NewRouter()

	chiRouter.Group(func(guarded chi.Router) {
		guarded.Use(VerifyJWT(tools, jwtauth.New("HS256", []byte(os.Getenv("TOKEN_SECRET")), nil)))
		guarded.Route("/v1", func(v1 chi.Router) {
			v1.Route("/streams", func(s chi.Router) {
				s.Route("/{id}", func(sid chi.Router) {
					sid.Get("/", streamController.GetStream)
				})
			})
		})
	})

	ts := httptest.NewServer(chiRouter)
	defer ts.Close()

	if resp, _ := test_utilities.TestRequest(t, ts, "GET", "/v1/streams/5938b99cb6906eb1fbaf1f1e", nil, testToken); resp.StatusCode != 503 {
		t.Fatalf(fmt.Sprintf("%d was returned instead of 503", resp.StatusCode))
	}
}
func TestStreamController_GetStream_DatabaseDown(t *testing.T) {
	//Flushing redis cache before start of test
	err := test_utilities.FlushRedis()
	if err != nil {
		t.Error(err)
	}
	mongo, _ := test_utilities.GetMongoDBClient()
	tools := test_utilities.TestSetup()
	streamController := api.NewStreamController(mongo.Database(os.Getenv("MONGO_DB_NAME")), tools)
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	_ = mongo.Disconnect(ctx)

	testToken := test_utilities.GenerateFakeTestToken()
	chiRouter := chi.NewRouter()

	chiRouter.Group(func(guarded chi.Router) {
		guarded.Use(VerifyJWT(tools, jwtauth.New("HS256", []byte(os.Getenv("TOKEN_SECRET")), nil)))
		guarded.Route("/v1", func(v1 chi.Router) {
			v1.Route("/streams", func(s chi.Router) {
				s.Route("/{id}", func(sid chi.Router) {
					sid.Get("/", streamController.GetStream)
				})
			})
		})
	})

	ts := httptest.NewServer(chiRouter)
	defer ts.Close()

	if resp, _ := test_utilities.TestRequest(t, ts, "GET", "/v1/streams/5938b99cb6906eb1fbaf1f1c", nil, testToken); resp.StatusCode != 500 {
		t.Fatalf(fmt.Sprintf("%d", resp.StatusCode))
	}
}
func TestStreamController_GetStream_DatabaseNameMissing(t *testing.T) {
	//Flushing redis cache before start of test
	err := test_utilities.FlushRedis()
	if err != nil {
		t.Error(err)
	}
	mongo, _ := test_utilities.GetMongoDBClient()
	tools := test_utilities.TestSetup()

	streamController := api.NewStreamController(mongo.Database(os.Getenv("")), tools)
	testToken := test_utilities.GenerateFakeTestToken()
	dbnamemissing := chi.NewRouter()

	dbnamemissing.Group(func(guarded chi.Router) {
		guarded.Use(VerifyJWT(tools, jwtauth.New("HS256", []byte(os.Getenv("TOKEN_SECRET")), nil)))
		//guarded.Post("/refresh",usersController.Logout)
		guarded.Route("/v1", func(v1 chi.Router) {
			v1.Route("/streams", func(s chi.Router) {
				s.Route("/{id}", func(sid chi.Router) {
					sid.Get("/", streamController.GetStream)
				})
			})
		})
	})

	ts := httptest.NewServer(dbnamemissing)
	defer ts.Close()

	if resp, _ := test_utilities.TestRequest(t, ts, "GET", "/v1/streams/5938b99cb6906eb1fbaf1f1c", nil, testToken); resp.StatusCode != 500 {
		t.Fatalf(fmt.Sprintf("%d", resp.StatusCode))
	}
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
