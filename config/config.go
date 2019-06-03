package config

import (
	"context"
	"github.com/go-redis/redis"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"time"
)

type Tools struct {
	Cache  *redis.Client
	Logger *zap.Logger
}

var logger *zap.Logger

func SetupLoggerAndCacheAndMongo() (*mongo.Client, *Tools) {
	logger, _ = setUpLogger()
	mongoClient := setUpMongo()
	redisCache := setUpRedis()

	return mongoClient, &Tools{redisCache, logger}
}
func setUpLogger() (*zap.Logger, error) {
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
	return cfg.Build()
}
func setUpMongo() *mongo.Client {
	//Mongodb set up
	mongoURI := os.Getenv("MONGO_URI")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		logger.Fatal(err.Error())
	}

	return client
}
func setUpRedis() *redis.Client {
	//Redis client setup
	cache := redis.NewClient(&redis.Options{
		Addr:         os.Getenv("REDIS_ADDRESS"),
		Password:     os.Getenv("REDIS_PASSWORD"),
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})

	//Check connection to redis cache
	_, err := cache.Ping().Result()
	if err != nil {
		logger.Fatal("redis gave " + err.Error())
	}
	return cache
}
