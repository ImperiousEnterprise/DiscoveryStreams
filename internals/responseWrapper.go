package internals

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

//Custom Error Messages
var DBError = errors.New("database error (something wrong on our end)")
var AdsError = errors.New("ads url metadata error")
var RedisError = errors.New("caching error (something wrong on our end)")
var NoStreamError = errors.New("no such stream exists")
var DuplicateError = errors.New("email already in use")
var LoginError = errors.New("email or password was incorrect")
var TokenGenError = errors.New("failed to generate token")
var TokenNotValidError = errors.New("token no longer valid")

//Converts error array to json by
// returning {"error":[{"message": <error_message>}, {"message": <error_message>}]}
func errorArrayToJson(err []error, statusCode int) []byte {
	type Message struct {
		Text string `json:"message"`
	}
	var Errors struct {
		Status      int
		ErrMessages []Message `json:"errors"`
	}

	Errors.Status = statusCode
	for _, e := range err {
		Errors.ErrMessages = append(Errors.ErrMessages, Message{e.Error()})
	}
	errorMarshall, _ := json.Marshal(Errors)
	return errorMarshall
}

//Returns errors as json to clients by accepting either a
//single error or an array of errors
func RespondAsErrorJson(w http.ResponseWriter, statusCode int, errors ...interface{}) {
	var errorText []byte
	for _, err := range errors {
		switch e := err.(type) {
		case error:
			errorText = []byte(fmt.Sprintf(`{"error":{"status": %d, "message": "%s"}}`, statusCode, e.Error()))
		case []error:
			errorText = errorArrayToJson(e, statusCode)
		}
	}
	RespondAsJson(w, errorText, statusCode)
}

//Returns json to clients
func RespondAsJson(w http.ResponseWriter, json []byte, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(json)
}
