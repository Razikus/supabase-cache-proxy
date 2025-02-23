package postgrestcache

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type CachedResponse struct {
	Body       []byte              `json:"body"`
	StatusCode int                 `json:"status_code"`
	Headers    map[string][]string `json:"headers"`
}

type CacheResponseWriter struct {
	http.ResponseWriter
	buffer     *bytes.Buffer
	statusCode int
	headers    map[string][]string
}

func NewCacheResponseWriter(w http.ResponseWriter) *CacheResponseWriter {
	return &CacheResponseWriter{
		ResponseWriter: w,
		buffer:         &bytes.Buffer{},
		statusCode:     http.StatusOK,
		headers:        make(map[string][]string),
	}
}

func (w *CacheResponseWriter) Write(b []byte) (int, error) {
	w.buffer.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *CacheResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *CacheResponseWriter) Header() http.Header {
	h := w.ResponseWriter.Header()
	w.headers = h
	return h
}

func (w *CacheResponseWriter) ToCachedResponse() *CachedResponse {
	return &CachedResponse{
		Body:       w.buffer.Bytes(),
		StatusCode: w.statusCode,
		Headers:    w.headers,
	}
}

func (w *CacheResponseWriter) Serialize() ([]byte, error) {
	cached := w.ToCachedResponse()
	return json.Marshal(cached)
}

func DeserializeResponse(data []byte) (*CachedResponse, error) {
	var cached CachedResponse
	err := json.Unmarshal(data, &cached)
	return &cached, err
}

func (cr *CachedResponse) WriteTo(w http.ResponseWriter) error {
	for key, values := range cr.Headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(cr.StatusCode)

	_, err := w.Write(cr.Body)
	return err
}
