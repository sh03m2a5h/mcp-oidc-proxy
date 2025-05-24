package proxy

import (
	"bytes"
	"net/http"
)

// ResponseRecorder captures HTTP response for retry logic
type ResponseRecorder struct {
	StatusCode int
	HeaderMap  http.Header
	Body       *bytes.Buffer
	written    bool
}

// NewResponseRecorder creates a new response recorder
func NewResponseRecorder() *ResponseRecorder {
	return &ResponseRecorder{
		StatusCode: http.StatusOK,
		HeaderMap:  make(http.Header),
		Body:       &bytes.Buffer{},
	}
}

// WriteHeader implements http.ResponseWriter
func (r *ResponseRecorder) WriteHeader(statusCode int) {
	if !r.written {
		r.StatusCode = statusCode
		r.written = true
	}
}

// Write implements http.ResponseWriter
func (r *ResponseRecorder) Write(data []byte) (int, error) {
	if !r.written {
		r.WriteHeader(http.StatusOK)
	}
	return r.Body.Write(data)
}

// Header implements http.ResponseWriter
func (r *ResponseRecorder) Header() http.Header {
	return r.HeaderMap
}

// WriteTo writes the recorded response to the provided ResponseWriter
func (r *ResponseRecorder) WriteTo(w http.ResponseWriter) {
	// Copy headers
	for key, values := range r.HeaderMap {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Write status code
	w.WriteHeader(r.StatusCode)

	// Write body
	if r.Body.Len() > 0 {
		w.Write(r.Body.Bytes())
	}
}