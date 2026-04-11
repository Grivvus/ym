package handlers_test

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/handlers"
)

func TestWriteJSON_SetsStatusContentTypeAndBody(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	payload := struct {
		Message string `json:"message"`
	}{
		Message: "ok",
	}

	err := handlers.WriteJSON(recorder, http.StatusCreated, payload)
	if err != nil {
		t.Fatalf("write json returned error: %v", err)
	}

	if recorder.Code != http.StatusCreated {
		t.Fatalf("unexpected status code: got %d want %d", recorder.Code, http.StatusCreated)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("unexpected content type: got %q want %q", got, "application/json")
	}

	var body struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Message != payload.Message {
		t.Fatalf("unexpected body message: got %q want %q", body.Message, payload.Message)
	}
}

func TestWriteJSON_ReturnsEncodeErrorForUnsupportedValue(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()

	err := handlers.WriteJSON(
		recorder,
		http.StatusOK,
		map[string]float64{"value": math.NaN()},
	)
	if err == nil {
		t.Fatal("expected encode error, got nil")
	}

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("unexpected content type: got %q want %q", got, "application/json")
	}
}

func TestWriteError_WritesErrorResponse(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()

	err := handlers.WriteError(recorder, http.StatusBadRequest, errors.New("boom"))
	if err != nil {
		t.Fatalf("write error returned error: %v", err)
	}

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d want %d", recorder.Code, http.StatusBadRequest)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("unexpected content type: got %q want %q", got, "application/json")
	}

	var body api.ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Error != "boom" {
		t.Fatalf("unexpected error message: got %q want %q", body.Error, "boom")
	}
}

func TestFormValueToBool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "lower true", value: "true", want: true},
		{name: "capitalized true", value: "True", want: true},
		{name: "upper true is false", value: "TRUE", want: false},
		{name: "false", value: "false", want: false},
		{name: "empty", value: "", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := handlers.FormValueToBool(tt.value)
			if got != tt.want {
				t.Fatalf("unexpected bool value: got %v want %v", got, tt.want)
			}
		})
	}
}
