package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewResponseRecorder(t *testing.T) {
	recorder := NewResponseRecorder()
	
	assert.NotNil(t, recorder)
	assert.Equal(t, http.StatusOK, recorder.StatusCode)
	assert.NotNil(t, recorder.HeaderMap)
	assert.NotNil(t, recorder.Body)
	assert.False(t, recorder.written)
}

func TestResponseRecorder_WriteHeader(t *testing.T) {
	recorder := NewResponseRecorder()
	
	// First write should work
	recorder.WriteHeader(http.StatusCreated)
	assert.Equal(t, http.StatusCreated, recorder.StatusCode)
	assert.True(t, recorder.written)
	
	// Subsequent writes should be ignored
	recorder.WriteHeader(http.StatusBadRequest)
	assert.Equal(t, http.StatusCreated, recorder.StatusCode)
}

func TestResponseRecorder_Write(t *testing.T) {
	recorder := NewResponseRecorder()
	
	// Write without WriteHeader should default to 200
	testContent := []byte("test content")
	n, err := recorder.Write(testContent)
	assert.NoError(t, err)
	assert.Equal(t, len(testContent), n)
	assert.Equal(t, http.StatusOK, recorder.StatusCode)
	assert.True(t, recorder.written)
	assert.Equal(t, "test content", recorder.Body.String())
}

func TestResponseRecorder_Header(t *testing.T) {
	recorder := NewResponseRecorder()
	
	recorder.Header().Set("Content-Type", "application/json")
	recorder.Header().Set("X-Custom-Header", "test")
	
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	assert.Equal(t, "test", recorder.Header().Get("X-Custom-Header"))
}

func TestResponseRecorder_WriteTo(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(recorder *ResponseRecorder)
		wantStatus int
		wantBody   string
		wantHeaders map[string]string
	}{
		{
			name: "Write complete response",
			setup: func(recorder *ResponseRecorder) {
				recorder.Header().Set("Content-Type", "text/plain")
				recorder.Header().Set("X-Test", "value")
				recorder.WriteHeader(http.StatusCreated)
				recorder.Write([]byte("test response"))
			},
			wantStatus: http.StatusCreated,
			wantBody:   "test response",
			wantHeaders: map[string]string{
				"Content-Type": "text/plain",
				"X-Test":       "value",
			},
		},
		{
			name: "Write with multiple header values",
			setup: func(recorder *ResponseRecorder) {
				recorder.Header().Add("X-Multi", "value1")
				recorder.Header().Add("X-Multi", "value2")
				recorder.WriteHeader(http.StatusOK)
				recorder.Write([]byte("multi header test"))
			},
			wantStatus: http.StatusOK,
			wantBody:   "multi header test",
			wantHeaders: map[string]string{
				"X-Multi": "value1",
			},
		},
		{
			name: "Empty response",
			setup: func(recorder *ResponseRecorder) {
				recorder.WriteHeader(http.StatusNoContent)
			},
			wantStatus: http.StatusNoContent,
			wantBody:   "",
			wantHeaders: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create recorder and apply setup
			recorder := NewResponseRecorder()
			tt.setup(recorder)
			
			// Create test response writer to write to
			w := httptest.NewRecorder()
			
			// Execute WriteTo
			recorder.WriteTo(w)
			
			// Verify status code
			assert.Equal(t, tt.wantStatus, w.Code)
			
			// Verify body
			assert.Equal(t, tt.wantBody, w.Body.String())
			
			// Verify headers
			for key, value := range tt.wantHeaders {
				assert.Equal(t, value, w.Header().Get(key))
			}
		})
	}
}