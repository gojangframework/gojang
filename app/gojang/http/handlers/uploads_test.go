package handlers

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func multipartRequest(t *testing.T, fieldName, fileName string, fileBytes int) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := part.Write(bytes.Repeat([]byte("a"), fileBytes)); err != nil {
		t.Fatalf("part.Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestParseMultipartFormLimited_AllowsRequestUnderLimit(t *testing.T) {
	req := multipartRequest(t, "file", "small.txt", 128)
	rec := httptest.NewRecorder()

	if err := ParseMultipartFormLimited(rec, req, 4096); err != nil {
		t.Fatalf("ParseMultipartFormLimited() error = %v", err)
	}

	file, header, err := req.FormFile("file")
	if err != nil {
		t.Fatalf("FormFile() error = %v", err)
	}
	defer file.Close()

	if header.Filename != "small.txt" {
		t.Fatalf("filename = %q, want %q", header.Filename, "small.txt")
	}
}

func TestParseMultipartFormLimited_ReturnsMaxUploadSizeError(t *testing.T) {
	req := multipartRequest(t, "file", "large.txt", 4096)
	rec := httptest.NewRecorder()

	err := ParseMultipartFormLimited(rec, req, 512)
	if err == nil {
		t.Fatal("ParseMultipartFormLimited() error = nil, want MaxUploadSizeError")
	}
	if !IsMaxUploadSizeError(err) {
		t.Fatalf("IsMaxUploadSizeError() = false for %T: %v", err, err)
	}

	var sizeErr *MaxUploadSizeError
	if !errors.As(err, &sizeErr) {
		t.Fatalf("errors.As(MaxUploadSizeError) = false")
	}
	if sizeErr.Limit != 512 {
		t.Fatalf("Limit = %d, want 512", sizeErr.Limit)
	}
}

