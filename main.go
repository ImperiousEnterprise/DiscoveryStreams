package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/dimiro1/banner/autoload"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

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

type StreamController struct {
	DB     *mongo.Client
	DBname string
	logger *zap.SugaredLogger
}

func main() {
	mongoURI := os.Getenv("MONGO_URI")
	mongoName := os.Getenv("MONGO_NAME")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}

	logger, _ := zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	streamController := NewStreamController(client, mongoName, sugar)
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(Logging(sugar))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Route("/v1", func(v1 chi.Router) {
		v1.Route("/streams", func(s chi.Router) {
			s.Route("/{id}", func(sid chi.Router) {
				sid.Use(streamController.StreamCtx)
				sid.Get("/", streamController.GetStream)
			})
		})
	})

	if err := http.ListenAndServe(":7000", r); err != nil {
		client.Disconnect(ctx)
		log.Fatal(err)
	}
}
func NewStreamController(mongo *mongo.Client, name string, log *zap.SugaredLogger) *StreamController {
	return &StreamController{
		DB:     mongo,
		DBname: name,
		logger: log,
	}
}
func Logging(l *zap.SugaredLogger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			defer func() {
				l.Infow("Served",
					"ip_address", r.RemoteAddr,
					"protocol_version", r.Proto,
					"request_method", r.Method,
					"path", r.URL.Path,
					"status", ww.Status(),
					"size", ww.BytesWritten(),
					"reqId", middleware.GetReqID(r.Context()))
			}()

			next.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}
}

func (jc *StreamController) StreamCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		streamID := chi.URLParam(r, "id")
		collection := jc.DB.Database(jc.DBname).Collection("streams")
		ctx, _ := context.WithTimeout(r.Context(), 5*time.Second)
		size, err := collection.CountDocuments(ctx, bson.M{"_id": streamID})
		if err != nil {
			jc.logger.Error(err)
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		} else if size == 0 {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (jc *StreamController) GetStream(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, "id")
	collection := jc.DB.Database(jc.DBname).Collection("streams")
	ctx, _ := context.WithTimeout(r.Context(), 5*time.Second)

	var stream Stream
	err := collection.FindOne(ctx, bson.M{"_id": streamID}).Decode(&stream)
	if err != nil {
		jc.logger.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	ads, err := getAds(os.Getenv("ADS_URL")+streamID, jc.logger)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
	stream.Ads = ads
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(stream.toJson())
}
func (s Stream) toJson() json.RawMessage {
	streamJson, _ := json.MarshalIndent(s, "", "  ")
	streamStr := string(streamJson)
	idEnd := strings.Index(streamStr, ",")
	streamUrlEnd := strings.Index(streamStr[idEnd+1:], ",")
	ads := strings.Index(streamStr, "ads")
	updatedCaption := strings.Replace(streamStr[idEnd+streamUrlEnd+1:ads], "/", `\/`, -1)
	return json.RawMessage(streamStr[:idEnd+streamUrlEnd+1] + updatedCaption + streamStr[ads:])
}
func getAds(url string, logger *zap.SugaredLogger) (json.RawMessage, error) {
	var ads json.RawMessage

	spaceClient := http.Client{
		Timeout: time.Second * 2,
	}

	res, getErr := spaceClient.Get(url)
	if getErr != nil {
		logger.Error(getErr)
		return ads, getErr
	} else if res.StatusCode >= 400 {
		getErr = errors.New(fmt.Sprintf("Returned %d from %s", res.StatusCode, url))
		logger.Error(getErr)
		return ads, getErr
	}

	err := json.NewDecoder(res.Body).Decode(&ads)
	if err != nil {
		logger.Error(err)
		return ads, err
	}

	return ads, nil
}
