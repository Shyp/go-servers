package server

import "net/http"

// Error is an error you return from your HTTP API.
type Error struct {
	Title      string `json:"title"`
	Id         string `json:"id"`
	Detail     string `json:"detail,omitempty"`
	Instance   string `json:"instance,omitempty"`
	Type       string `json:"type,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
}

func (e *Error) Error() string {
	return e.Title
}

func new404(r *http.Request) Error {
	return Error{
		Title:      "Resource not found",
		Id:         "not_found",
		Instance:   r.URL.Path,
		StatusCode: 404,
	}
}

func new405(r *http.Request) Error {
	return Error{
		Title:      "Method not allowed",
		Id:         "method_not_allowed",
		Instance:   r.URL.Path,
		StatusCode: http.StatusMethodNotAllowed,
	}
}
