package api

import (
	"DiscoveryStreams/config"
	"DiscoveryStreams/internals"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-redis/redis"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"net/http"
	"os"
	"strings"
	"time"
)

//Struct to hold Stream data that's set to return to client
type Stream struct {
	ID        string `json:"id" bson:"_id"`
	StreamURL string `json:"streamUrl" bson:"streamUrl"`
	Captions  struct {
		Vtt struct {
			En string `json:"en" bson:"en"`
		} `json:"vtt" bson:"vtt"`
		Scc struct {
			En string `json:"en" bson:"en"`
		} `json:"scc" bson:"scc"`
	} `json:"captions" bson:"captions"`
	Ads json.RawMessage `json:"ads"`
}

//Struct to give Stream API endpoints
//access to mongo, logging, and cache
type StreamController struct {
	streamCollection *mongo.Collection
	*config.Tools
}

func NewStreamController(mongo *mongo.Database, tools *config.Tools) *StreamController {
	collection := mongo.Collection("streams")
	return &StreamController{
		streamCollection: collection,
		Tools:            tools,
	}
}

//Stream Controller's Get method that's responsible for getting stream id
// data from mongo and ad url endpoint.
func (s *StreamController) GetStream(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, "id")
	hit, e := s.Cache.Get(streamID).Result()
	if hit != "" {
		internals.RespondAsJson(w, []byte(hit), http.StatusOK)
		return
	} else if e != nil && e != redis.Nil {
		s.Logger.Error(e.Error(), zap.String("reqId", middleware.GetReqID(r.Context())))
	}

	var stream Stream
	ctx, _ := context.WithTimeout(r.Context(), 5*time.Second)
	err := s.streamCollection.FindOne(ctx, bson.M{"_id": streamID}).Decode(&stream)
	if err == mongo.ErrNoDocuments {
		internals.RespondAsErrorJson(w, http.StatusNotFound, internals.NoStreamError)
		return
	} else if err != nil {
		s.Logger.Error("mongo returned "+err.Error(), zap.String("reqId", middleware.GetReqID(r.Context())))
		internals.RespondAsErrorJson(w, http.StatusInternalServerError, internals.DBError)
		return
	}

	stream.Ads, err = getAds(os.Getenv("ADS_URL") + streamID)
	if err != nil {
		s.Logger.Error(err.Error(), zap.String("reqId", middleware.GetReqID(r.Context())))
		internals.RespondAsErrorJson(w, http.StatusServiceUnavailable, internals.AdsError)
		return
	}

	streamJson := stream.toJson()
	err = s.Cache.Set(streamID, []byte(streamJson), 0).Err()
	if err != nil {
		s.Logger.Error(e.Error(), zap.String("reqId", middleware.GetReqID(r.Context())))
	}
	internals.RespondAsJson(w, streamJson, http.StatusOK)
}

//Converts Stream struct to JSON then updates caption's slash(/) to encoded slash(\/).
//This is done because go's marshalling does not encode / to \/.
func (s Stream) toJson() json.RawMessage {
	streamJson, _ := json.MarshalIndent(s, "", "  ")
	streamStr := string(streamJson)
	idEnd := strings.Index(streamStr, ",")
	streamUrlEnd := strings.Index(streamStr[idEnd+1:], ",")
	ads := strings.Index(streamStr, "ads")
	updatedCaption := strings.Replace(streamStr[idEnd+streamUrlEnd+1:ads], "/", `\/`, -1)
	return json.RawMessage(streamStr[:idEnd+streamUrlEnd+1] + updatedCaption + streamStr[ads:])
}

//Uses ads url to fetch ad metadata for a streamID and stores the response
// as raw json to be used in Streams.
func getAds(url string) (json.RawMessage, error) {
	var ads json.RawMessage

	spaceClient := http.Client{
		Timeout: time.Second * 3,
	}

	res, getErr := spaceClient.Get(url)
	if getErr != nil {
		return ads, getErr
	} else if res.StatusCode >= 400 {
		getErr = errors.New(fmt.Sprintf("Returned %d from %s", res.StatusCode, url))
		return ads, getErr
	}
	err := json.NewDecoder(res.Body).Decode(&ads)
	if err != nil {
		return ads, err
	}

	return ads, nil
}
