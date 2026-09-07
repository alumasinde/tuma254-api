package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

const DefaultMaxBodyBytes int64 = 1 << 20

// DecodeJSONStrict enforces a bounded request body, a JSON content type,
// unknown-field rejection, and exactly one JSON value.
func DecodeJSONStrict(w http.ResponseWriter, r *http.Request, dst any) error {
	if r.Body == nil {
		return errors.New("request body required")
	}
	if contentType := strings.TrimSpace(r.Header.Get("Content-Type")); contentType != "" {
		mediaType := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
		if mediaType != "application/json" {
			return errors.New("content type must be application/json")
		}
	}
	r.Body = http.MaxBytesReader(w, r.Body, DefaultMaxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must contain exactly one JSON value")
	}
	return nil
}
