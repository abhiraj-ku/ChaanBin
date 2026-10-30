package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/abhiraj-ku/ChaanBin/internal/store"
)

type fakeLogSearcher struct {
	results []store.SearchResults
	err     error
	params  store.SearchParams
	calls   int
}

func (f *fakeLogSearcher) SearchSimilarLogs(_ context.Context, params store.SearchParams) ([]store.SearchResults, error) {
	f.calls++
	f.params = params
	return f.results, f.err
}

type fakeQueryEmbedder struct {
	vector []float32
	err    error
	query  string
}

func (f *fakeQueryEmbedder) GetQueryPrefix(_ context.Context, query string) ([]float32, error) {
	f.query = query
	return f.vector, f.err
}

func TestSearchLogsRejectsNonPost(t *testing.T) {
	handler := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search", nil)
	rec := httptest.NewRecorder()

	handler.SearchLogs(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestSearchLogsRejectsInvalidJSON(t *testing.T) {
	handler := &Handler{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewBufferString("{"))
	rec := httptest.NewRecorder()

	handler.SearchLogs(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestSearchLogsReturnsEmbeddingError(t *testing.T) {
	embedder := &fakeQueryEmbedder{err: errors.New("embedding unavailable")}
	handler := &Handler{Embedder: embedder, Repo: &fakeLogSearcher{}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewBufferString(`{"query":"error"}`))
	rec := httptest.NewRecorder()

	handler.SearchLogs(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestSearchLogsReturnsRepositoryError(t *testing.T) {
	embedder := &fakeQueryEmbedder{vector: []float32{0.1, 0.2}}
	repo := &fakeLogSearcher{err: errors.New("database unavailable")}
	handler := &Handler{Embedder: embedder, Repo: repo}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewBufferString(`{"query":"error"}`))
	rec := httptest.NewRecorder()

	handler.SearchLogs(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestSearchLogsUsesDefaultsAndReturnsResults(t *testing.T) {
	embedder := &fakeQueryEmbedder{vector: []float32{0.1, 0.2}}
	repo := &fakeLogSearcher{
		results: []store.SearchResults{{
			ID:         1,
			Service:    "api",
			Severity:   "error",
			Message:    "request failed",
			TimeStamp:  time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC),
			Similarity: 0.95,
		}},
	}
	handler := &Handler{Embedder: embedder, Repo: repo}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewBufferString(`{"query":"request failed"}`))
	rec := httptest.NewRecorder()

	handler.SearchLogs(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if embedder.query != "request failed" {
		t.Fatalf("embedding query = %q, want %q", embedder.query, "request failed")
	}
	if repo.params.DaysLookback != 7 || repo.params.Limit != 10 {
		t.Fatalf("params = %+v, want defaults days=7 limit=10", repo.params)
	}
	if len(repo.params.QueryVector) != 2 {
		t.Fatalf("query vector length = %d, want 2", len(repo.params.QueryVector))
	}

	var results []store.SearchResults
	if err := json.NewDecoder(rec.Body).Decode(&results); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(results) != 1 || results[0].ID != 1 {
		t.Fatalf("results = %+v, want one result with ID 1", results)
	}
}

func TestSearchLogsPassesExplicitFilters(t *testing.T) {
	severity := "warn"
	embedder := &fakeQueryEmbedder{vector: []float32{0.3}}
	repo := &fakeLogSearcher{}
	handler := &Handler{Embedder: embedder, Repo: repo}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewBufferString(
		`{"query":"slow","severity":"warn","days_lookback":30,"limit":25}`,
	))
	rec := httptest.NewRecorder()

	handler.SearchLogs(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if repo.params.DaysLookback != 30 || repo.params.Limit != 25 || repo.params.Severity == nil || *repo.params.Severity != severity {
		t.Fatalf("params = %+v, want explicit filters", repo.params)
	}
}

func TestSearchLogsEncodesEmptyResultsAsArray(t *testing.T) {
	handler := &Handler{
		Embedder: &fakeQueryEmbedder{vector: []float32{0.1}},
		Repo:     &fakeLogSearcher{},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewBufferString(`{"query":"none"}`))
	rec := httptest.NewRecorder()

	handler.SearchLogs(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != "[]\n" {
		t.Fatalf("body = %q, want empty JSON array", got)
	}
}
