package main

import (
	"DiscoveryStreams/api"
	"DiscoveryStreams/config"
	"DiscoveryStreams/internals"
	"context"
	"fmt"
	_ "github.com/dimiro1/banner/autoload"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/jwtauth"
	"github.com/go-redis/redis"
	"go.uber.org/zap"
	"net/http"
	"os"
	"time"
)

func main() {
	tokenAuth := jwtauth.New("HS256", []byte(os.Getenv("TOKEN_SECRET")), nil)
	port := fmt.Sprintf(":%s", os.Getenv("PORT"))
	//set up routes
	mongo, tools := config.SetupLoggerAndCacheAndMongo()
	streamController := api.NewStreamController(mongo.Database(os.Getenv("MONGO_DB_NAME")), tools)
	usersController := api.NewUsersController(mongo.Database(os.Getenv("MONGO_DB_NAME")), tools)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(LogRequests(tools.Logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Post("/login", usersController.Login)
	r.Post("/signup", usersController.Signup)

	//JWT protected routes
	r.Group(func(guarded chi.Router) {
		guarded.Use(VerifyJWT(tools, tokenAuth))
		guarded.Post("/logout", usersController.Logout)
		guarded.Route("/v1", func(v1 chi.Router) {
			v1.Route("/streams", func(s chi.Router) {
				s.Route("/{id}", func(sid chi.Router) {
					sid.Get("/", streamController.GetStream)
				})
			})
		})
	})

	defer tools.Logger.Sync() //clear logger buffer
	if err := http.ListenAndServe(port, r); err != nil {
		ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
		mongo.Disconnect(ctx)
		tools.Cache.Close()
		tools.Logger.Fatal(err.Error())
	}
}

func VerifyJWT(tools *config.Tools, token *jwtauth.JWTAuth) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			tkn, err := jwtauth.VerifyRequest(token, r, jwtauth.TokenFromHeader, jwtauth.TokenFromQuery, jwtauth.TokenFromCookie)
			if err != nil {
				//Removed the word 'jwtauth: ' from error string
				internals.RespondAsErrorJson(w, http.StatusUnauthorized, err.Error()[9:])
				return
			}

			//check redis blacklist
			exists, e := tools.Cache.Exists(tkn.Raw).Result()
			if exists == 1 {
				internals.RespondAsErrorJson(w, http.StatusUnauthorized, internals.TokenNotValidError)
				return
			} else if e != nil && e != redis.Nil {
				tools.Logger.Error(e.Error(), zap.String("reqId", middleware.GetReqID(r.Context())))
				internals.RespondAsErrorJson(w, http.StatusInternalServerError, internals.RedisError)
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "Token", tkn)))
		}
		return http.HandlerFunc(fn)
	}
}
func LogRequests(l *zap.Logger) func(next http.Handler) http.Handler {
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
