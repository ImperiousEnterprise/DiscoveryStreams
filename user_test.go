package main

import (
	"DiscoveryStreams/api"
	"DiscoveryStreams/test_utilities"
	"bytes"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/jwtauth"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestUsersController_SignUp_Good(t *testing.T) {
	mongo, err := test_utilities.GetMongoDBClient()
	if err != nil {
		t.Fatal(err)
	}
	tools := test_utilities.TestSetup()
	usersController := api.NewUsersController(mongo.Database(os.Getenv("MONGO_DB_NAME")), tools)

	chiRouter := chi.NewRouter()
	chiRouter.Post("/signup", usersController.Signup)
	ts := httptest.NewServer(chiRouter)
	defer ts.Close()
	signUpBody := []byte(`{"email":"test@email.com","firstname":"Test", "lastname":"User", "password":"test12"}`)
	if resp, _ := test_utilities.TestRequest(t, ts, "POST", "/signup", bytes.NewReader(signUpBody), ""); resp.StatusCode != http.StatusCreated {
		t.Fatalf(fmt.Sprintf("%d was returned instead of 200", resp.StatusCode))
	}
}

func TestUsersController_SignUp_Bad(t *testing.T) {
	mongo, err := test_utilities.GetMongoDBClient()
	if err != nil {
		t.Fatal(err)
	}
	tools := test_utilities.TestSetup()
	usersController := api.NewUsersController(mongo.Database(os.Getenv("MONGO_DB_NAME")), tools)

	chiRouter := chi.NewRouter()
	chiRouter.Post("/signup", usersController.Signup)
	ts := httptest.NewServer(chiRouter)
	defer ts.Close()
	signUpBody := []byte(`{"email":"","firstname":"", "lastname":"", "password":""}`)
	if resp, _ := test_utilities.TestRequest(t, ts, "POST", "/signup", bytes.NewReader(signUpBody), ""); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf(fmt.Sprintf("%d was returned instead of 400", resp.StatusCode))
	}

	signUpBody = []byte(`{"email":"cooll@example.com","firstname":"s", "lastname":"a", "password":"no"}`)
	if resp, _ := test_utilities.TestRequest(t, ts, "POST", "/signup", bytes.NewReader(signUpBody), ""); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf(fmt.Sprintf("%d was returned instead of 400", resp.StatusCode))
	}
}

func TestUsersController_Login_ValidCredentials(t *testing.T) {
	mongo, err := test_utilities.GetMongoDBClient()
	if err != nil {
		t.Fatal(err)
	}
	tools := test_utilities.TestSetup()
	usersController := api.NewUsersController(mongo.Database(os.Getenv("MONGO_DB_NAME")), tools)

	chiRouter := chi.NewRouter()
	chiRouter.Post("/signup", usersController.Signup)
	chiRouter.Post("/login", usersController.Login)
	ts := httptest.NewServer(chiRouter)
	defer ts.Close()

	signUpBody := []byte(`{"email":"test@example.com","firstname":"Test", "lastname":"User", "password":"test12"}`)
	_, _ = test_utilities.TestRequest(t, ts, "POST", "/signup", bytes.NewReader(signUpBody), "")

	loginBody := []byte(`{"email":"test@example.com", "password":"test12"}`)
	if resp, _ := test_utilities.TestRequest(t, ts, "POST", "/login", bytes.NewReader(loginBody), ""); resp.StatusCode != http.StatusOK {
		t.Fatalf(fmt.Sprintf("%d was returned instead of 200", resp.StatusCode))
	}

}

func TestUsersController_Login_InvalidCredentials(t *testing.T) {
	mongo, err := test_utilities.GetMongoDBClient()
	if err != nil {
		t.Fatal(err)
	}
	tools := test_utilities.TestSetup()
	usersController := api.NewUsersController(mongo.Database(os.Getenv("MONGO_DB_NAME")), tools)

	chiRouter := chi.NewRouter()
	chiRouter.Post("/signup", usersController.Signup)
	chiRouter.Post("/login", usersController.Login)
	ts := httptest.NewServer(chiRouter)
	defer ts.Close()

	signUpBody := []byte(`{"email":"test2@email.com","firstname":"Test2", "lastname":"User", "password":"test12"}`)
	_, _ = test_utilities.TestRequest(t, ts, "POST", "/signup", bytes.NewReader(signUpBody), "")

	loginBody := []byte(`{"email":"test12@example.com", "password":"test12"}`)
	if resp, _ := test_utilities.TestRequest(t, ts, "POST", "/login", bytes.NewReader(loginBody), ""); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf(fmt.Sprintf("%d was returned instead of 401", resp.StatusCode))
	}

}

func TestUsersController_LogOut_Good(t *testing.T) {
	mongo, err := test_utilities.GetMongoDBClient()
	if err != nil {
		t.Fatal(err)
	}
	tools := test_utilities.TestSetup()
	usersController := api.NewUsersController(mongo.Database(os.Getenv("MONGO_DB_NAME")), tools)

	tokenAuth := jwtauth.New("HS256", []byte(os.Getenv("TOKEN_SECRET")), nil)

	chiRouter := chi.NewRouter()
	chiRouter.Post("/signup", usersController.Signup)
	chiRouter.Post("/login", usersController.Login)

	chiRouter.Group(func(guarded chi.Router) {
		guarded.Use(VerifyJWT(tools, tokenAuth))
		guarded.Delete("/logout", usersController.Logout)
	})

	ts := httptest.NewServer(chiRouter)
	defer ts.Close()

	signUpBody := []byte(`{"email":"test21@example.com","firstname":"Test2", "lastname":"User", "password":"test12"}`)
	_, _ = test_utilities.TestRequest(t, ts, "POST", "/signup", bytes.NewReader(signUpBody), "")

	loginBody := []byte(`{"email":"test21@example.com", "password":"test12"}`)
	resp, _ := test_utilities.TestRequest(t, ts, "POST", "/login", bytes.NewReader(loginBody), "")

	tkn := resp.Header.Get("Authorization")
	removeBearer := tkn[7:]

	if logoutResp, body := test_utilities.TestRequest(t, ts, "DELETE", "/logout", nil, removeBearer); logoutResp.StatusCode != http.StatusOK {
		t.Fatalf(fmt.Sprintf("%d was returned instead of 200 with %s", logoutResp.StatusCode, body))
	}
}
func TestUsersController_ReuseToken_After_Logout(t *testing.T) {
	mongo, err := test_utilities.GetMongoDBClient()
	if err != nil {
		t.Fatal(err)
	}
	tools := test_utilities.TestSetup()
	usersController := api.NewUsersController(mongo.Database(os.Getenv("MONGO_DB_NAME")), tools)

	tokenAuth := jwtauth.New("HS256", []byte(os.Getenv("TOKEN_SECRET")), nil)

	chiRouter := chi.NewRouter()
	chiRouter.Post("/signup", usersController.Signup)
	chiRouter.Post("/login", usersController.Login)

	chiRouter.Group(func(guarded chi.Router) {
		guarded.Use(VerifyJWT(tools, tokenAuth))
		guarded.Delete("/logout", usersController.Logout)
	})

	ts := httptest.NewServer(chiRouter)
	defer ts.Close()

	signUpBody := []byte(`{"email":"test213@example.com","firstname":"Test213", "lastname":"User", "password":"test12"}`)
	_, _ = test_utilities.TestRequest(t, ts, "POST", "/signup", bytes.NewReader(signUpBody), "")

	loginBody := []byte(`{"email":"test213@example.com", "password":"test12"}`)
	resp, _ := test_utilities.TestRequest(t, ts, "POST", "/login", bytes.NewReader(loginBody), "")

	tkn := resp.Header.Get("Authorization")
	removeBearer := tkn[7:]

	if logoutResp, _ := test_utilities.TestRequest(t, ts, "DELETE", "/logout", nil, removeBearer); logoutResp.StatusCode != http.StatusOK {
		t.Fatalf(fmt.Sprintf("%d was returned instead of 200", logoutResp.StatusCode))
	}

	if logoutResp, _ := test_utilities.TestRequest(t, ts, "DELETE", "/logout", nil, removeBearer); logoutResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf(fmt.Sprintf("%d was returned instead of 401", logoutResp.StatusCode))
	}
}

func TestUsersController_LogOut_NoToken(t *testing.T) {
	mongo, err := test_utilities.GetMongoDBClient()
	if err != nil {
		t.Fatal(err)
	}
	tools := test_utilities.TestSetup()
	usersController := api.NewUsersController(mongo.Database(os.Getenv("MONGO_DB_NAME")), tools)

	tokenAuth := jwtauth.New("HS256", []byte(os.Getenv("TOKEN_SECRET")), nil)

	chiRouter := chi.NewRouter()
	chiRouter.Group(func(guarded chi.Router) {
		guarded.Use(VerifyJWT(tools, tokenAuth))
		guarded.Delete("/logout", usersController.Logout)
	})

	ts := httptest.NewServer(chiRouter)
	defer ts.Close()

	if logoutResp, _ := test_utilities.TestRequest(t, ts, "DELETE", "/logout", nil, ""); logoutResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf(fmt.Sprintf("%d was returned instead of 401", logoutResp.StatusCode))
	}
}
