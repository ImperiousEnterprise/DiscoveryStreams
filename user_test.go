package main

import (
	"DiscoveryStreams/api"
	"DiscoveryStreams/config"
	"encoding/json"
	"github.com/go-chi/chi"
	"github.com/go-chi/jwtauth"
	"github.com/go-redis/redis"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestUsersController_SignUp_Good(t *testing.T) {
	chiRouter := chi.NewRouter()
	chiRouter.Post("/signup", usersController.Signup)
	ts := httptest.NewServer(chiRouter)
	defer ts.Close()

	if _, body := testRequest(t, ts, "POST", "/v1/streams/5938b99cb6906eb1fbaf1f1c", nil); body != goodCase {
		t.Fatalf(body)
	}
}
