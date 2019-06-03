package api

import (
	"DiscoveryStreams/config"
	"DiscoveryStreams/internals"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi/middleware"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"net/http"
	"os"
	"regexp"
	"time"
)

type User struct {
	Email     string `json:"email" bson:"email"`
	FirstName string `json:"firstname" bson:"firstname"`
	LastName  string `json:"lastname" bson:"lastname"`
	Password  string `json:"password" bson:"password"`
}

type UsersController struct {
	userCollection *mongo.Collection
	*config.Tools
}

func NewUsersController(mongo *mongo.Database, tools *config.Tools) *UsersController {
	collection := mongo.Collection("users")
	return &UsersController{
		userCollection: collection,
		Tools:          tools,
	}
}
func (u *User) validate() []error {
	var error []error
	emailRegex := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)
	if u.Email == "" {
		error = append(error, errors.New("email is required"))
	} else if !emailRegex.MatchString(u.Email) {
		error = append(error, errors.New("invalid email format"))
	}

	if u.FirstName == "" {
		error = append(error, errors.New("first name is required"))
	}

	if u.LastName == "" {
		error = append(error, errors.New("last name is required"))
	}

	if u.Password == "" {
		error = append(error, errors.New("password is required"))
	} else if len(u.Password) < 6 {
		error = append(error, errors.New("password needs to be more than 6 characters"))
	}

	return error
}
func (u *UsersController) Signup(w http.ResponseWriter, r *http.Request) {
	var user User
	_ = json.NewDecoder(r.Body).Decode(&user)

	errs := user.validate()
	if len(errs) != 0 {
		internals.RespondAsErrorJson(w, http.StatusBadRequest, errs)
		return
	}

	user.Password = hashPassword(user.Password)
	ctx, _ := context.WithTimeout(r.Context(), 5*time.Second)
	_, err := u.userCollection.InsertOne(ctx, user)
	if _, ok := err.(mongo.WriteException); ok {
		internals.RespondAsErrorJson(w, http.StatusBadRequest, internals.DuplicateError)
		return
	} else if err != nil {
		u.Logger.Error(err.Error(), zap.String("reqId", middleware.GetReqID(r.Context())))
		internals.RespondAsErrorJson(w, http.StatusServiceUnavailable, err)
		return
	}

	internals.RespondAsJson(w, nil, http.StatusCreated)
}

func (u *UsersController) Login(w http.ResponseWriter, r *http.Request) {

	var user User
	_ = json.NewDecoder(r.Body).Decode(&user)
	user.Password = hashPassword(user.Password)

	ctx, _ := context.WithTimeout(r.Context(), 5*time.Second)
	err := u.userCollection.FindOne(ctx, bson.M{"email": user.Email, "password": user.Password}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		internals.RespondAsErrorJson(w, http.StatusUnauthorized, internals.LoginError)
		return
	} else if err != nil {
		u.Logger.Error(err.Error(), zap.String("reqId", middleware.GetReqID(r.Context())))
		internals.RespondAsErrorJson(w, http.StatusInternalServerError, internals.DBError)
		return
	}

	tokenAuth, err := u.generateToken(user)
	if err != nil {
		u.Logger.Error(err.Error(), zap.String("reqId", middleware.GetReqID(r.Context())))
		internals.RespondAsErrorJson(w, http.StatusInternalServerError, internals.TokenGenError)
		return
	}
	w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", tokenAuth))

	internals.RespondAsJson(w, nil, http.StatusOK)

}

func (u *UsersController) Logout(w http.ResponseWriter, r *http.Request) {
	tkn := r.Context().Value("Token").(*jwt.Token)
	claims := &jwt.StandardClaims{}
	jwtKey := []byte(os.Getenv("TOKEN_SECRET"))
	_, _ = jwt.ParseWithClaims(tkn.Raw, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	timeLeft := time.Unix(claims.ExpiresAt, 0).Sub(time.Unix(time.Now().Unix(), 0))

	//Adding jwt to redis blacklist
	u.Cache.Set(tkn.Raw, true, timeLeft).Result()
	internals.RespondAsJson(w, nil, http.StatusOK)

}
func (u *UsersController) generateToken(user User) (string, error) {
	jti, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp":       time.Now().Add(time.Hour * 1).Unix(),
		"iat":       time.Now().Unix(),
		"email":     user.Email,
		"firstname": user.FirstName,
		"lastname":  user.LastName,
		"wholename": user.FirstName + " " + user.LastName,
		"jti":       jti.String(),
	})

	tokenString, err := token.SignedString([]byte(os.Getenv("TOKEN_SECRET")))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
func hashPassword(password string) string {
	h := sha1.New()
	h.Write([]byte(password))
	sha1Hash := hex.EncodeToString(h.Sum(nil))
	return sha1Hash
}
