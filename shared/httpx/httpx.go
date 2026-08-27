// Package httpx holds the JSON request and response helpers shared by every
// service transport layer.
package httpx

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

// Error is a domain error carrying the HTTP status it should map to.
type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }

// NewError builds an Error. Code is a stable machine-readable string; Message
// is shown to the caller, so it must not leak internal detail.
func NewError(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

// Common constructors for the statuses services actually return.
func BadRequest(code, msg string) *Error   { return NewError(http.StatusBadRequest, code, msg) }
func Unauthorized(code, msg string) *Error { return NewError(http.StatusUnauthorized, code, msg) }
func Forbidden(code, msg string) *Error    { return NewError(http.StatusForbidden, code, msg) }
func NotFound(code, msg string) *Error     { return NewError(http.StatusNotFound, code, msg) }
func Conflict(code, msg string) *Error     { return NewError(http.StatusConflict, code, msg) }

// JSON writes v as a JSON response with the given status.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("httpx: encode response: %v", err)
	}
}

// NoContent writes an empty 204 response.
func NoContent(w http.ResponseWriter) { w.WriteHeader(http.StatusNoContent) }

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// Fail writes err as a JSON error response. Errors that are not *Error map to
// 500 and are logged in full, so internal detail never reaches the client.
func Fail(w http.ResponseWriter, err error) {
	var e *Error
	if !errors.As(err, &e) {
		log.Printf("httpx: internal error: %v", err)
		e = NewError(http.StatusInternalServerError, "internal", "something went wrong")
	}
	var body errorBody
	body.Error.Code = e.Code
	body.Error.Message = e.Message
	JSON(w, e.Status, body)
}

// Decode reads a JSON request body into v. Unknown fields are rejected so
// typos in a client payload surface immediately instead of being ignored.
func Decode(r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return BadRequest("invalid_body", "request body is not valid JSON: "+err.Error())
	}
	return nil
}
