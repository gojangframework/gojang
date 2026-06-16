package handlers

import (
	"errors"
	"fmt"
	"net/http"
)

// DefaultMaxUploadBytes is the default maximum multipart request body size.
const DefaultMaxUploadBytes int64 = 32 << 20

// ParseMultipartFormLimited caps the request body before parsing multipart form
// data. It returns a MaxUploadSizeError when the request body exceeds maxBytes.
func ParseMultipartFormLimited(w http.ResponseWriter, r *http.Request, maxBytes int64) error {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxUploadBytes
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	if err := r.ParseMultipartForm(maxBytes); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return &MaxUploadSizeError{Limit: maxBytes}
		}
		return err
	}

	return nil
}

// MaxUploadSizeError reports that an upload exceeded the configured limit.
type MaxUploadSizeError struct {
	Limit int64
}

func (e *MaxUploadSizeError) Error() string {
	return fmt.Sprintf("upload exceeds maximum size of %d bytes", e.Limit)
}

// IsMaxUploadSizeError reports whether err is a MaxUploadSizeError.
func IsMaxUploadSizeError(err error) bool {
	var sizeErr *MaxUploadSizeError
	return errors.As(err, &sizeErr)
}

