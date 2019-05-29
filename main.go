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
	"go.uber.org/zap/zapcore"
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
	streamCollection *mongo.Collection
	logger           *zap.Logger
}

func main() {
	//Logger Config
	cfg := zap.Config{
		Encoding:         "json",
		Level:            zap.NewAtomicLevelAt(zapcore.DebugLevel),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey: "message",

			LevelKey:    "level",
			EncodeLevel: zapcore.CapitalLevelEncoder,

			TimeKey:    "time",
			EncodeTime: zapcore.ISO8601TimeEncoder,

			CallerKey:    "caller",
			EncodeCaller: zapcore.ShortCallerEncoder,
		},
	}

	logger, _ := cfg.Build()
	defer logger.Sync() // flushes buffer, if any

	mongoURI := os.Getenv("MONGO_URI")
	mongoName := os.Getenv("MONGO_NAME")
	port := fmt.Sprintf(":%s", os.Getenv("PORT"))
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		logger.Fatal(err.Error())
	}

	streamController := NewStreamController(client, mongoName, logger)
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(Logging(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Post("/login", nil)
	r.Route("/v1", func(v1 chi.Router) {
		v1.Route("/streams", func(s chi.Router) {
			s.Route("/{id}", func(sid chi.Router) {
				sid.Use(streamController.StreamCtx)
				sid.Get("/", streamController.GetStream)
			})
		})
	})

	if err := http.ListenAndServe(port, r); err != nil {
		client.Disconnect(ctx)
		logger.Fatal(err.Error())
	}
}
func NewStreamController(mongo *mongo.Client, name string, log *zap.Logger) *StreamController {
	collection := mongo.Database(name).Collection("streams")
	return &StreamController{
		streamCollection: collection,
		logger:           log,
	}
}
func Logging(l *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			defer func() {
				l.Info("Served",
					zap.String("ip_address", r.RemoteAddr),
					zap.String("protocol_version", r.Proto),
					zap.String("request_method", r.Method),
					zap.String("path", r.URL.Path),
					zap.Int("status", ww.Status()),
					zap.Int("size", ww.BytesWritten()),
					zap.String("reqId", middleware.GetReqID(r.Context())))
			}()

			next.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}
}

func (s *StreamController) StreamCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		streamID := chi.URLParam(r, "id")
		ctx, _ := context.WithTimeout(r.Context(), 5*time.Second)
		size, err := s.streamCollection.CountDocuments(ctx, bson.M{"_id": streamID})
		if err != nil {
			s.logger.Error(err.Error())
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		} else if size == 0 {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *StreamController) GetStream(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, "id")

	var stream Stream
	ctx, _ := context.WithTimeout(r.Context(), 5*time.Second)
	err := s.streamCollection.FindOne(ctx, bson.M{"_id": streamID}).Decode(&stream)
	if err != nil {
		s.logger.Error(err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	ads, err := getAds(os.Getenv("ADS_URL")+streamID, s.logger)
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
func getAds(url string, logger *zap.Logger) (json.RawMessage, error) {
	var ads json.RawMessage

	spaceClient := http.Client{
		Timeout: time.Second * 2,
	}

	res, getErr := spaceClient.Get(url)
	if getErr != nil {
		logger.Error(getErr.Error())
		return ads, getErr
	} else if res.StatusCode >= 400 {
		getErr = errors.New(fmt.Sprintf("Returned %d from %s", res.StatusCode, url))
		logger.Error(getErr.Error())
		return ads, getErr
	}

	err := json.NewDecoder(res.Body).Decode(&ads)
	if err != nil {
		logger.Error(err.Error())
		return ads, err
	}

	return ads, nil
}
