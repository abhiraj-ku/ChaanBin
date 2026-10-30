package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/abhiraj-ku/ChaanBin/internal/store"
)

type LogSearcher interface {
	SearchSimilarLogs(context.Context, store.SearchParams) ([]store.SearchResults, error)
}

type QueryEmbedder interface {
	GetQueryPrefix(context.Context, string) ([]float32, error)
}

type SearchRequest struct {
	Query        string  `json:"query"`
	Severity     *string `json:"severity,omitempty"`
	DaysLookback int     `json:"days_lookback"`
	Limit        int     `json:"limit"`
}

type Handler struct {
	Repo     LogSearcher
	Embedder QueryEmbedder
}

func (h *Handler) SearchLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Mehthod not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json payload", http.StatusBadRequest)
		return
	}

	// limit requ count

	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 10
	}

	// default to last 7 days
	if req.DaysLookback <= 0 {
		req.DaysLookback = 7
	}

	queryVec, err := h.Embedder.GetQueryPrefix(r.Context(), req.Query)
	if err != nil {
		http.Error(w, "failed to generate search emebedding", http.StatusInternalServerError)
		return
	}

	params := store.SearchParams{
		QueryVector:  queryVec,
		Severity:     req.Severity,
		DaysLookback: req.DaysLookback,
		Limit:        req.Limit,
	}

	results, err := h.Repo.SearchSimilarLogs(r.Context(), params)
	if err != nil {
		http.Error(w, "search failed", http.StatusInternalServerError)
		return
	}

	// sned empty array instead of nill for json
	if results == nil {
		results = []store.SearchResults{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

}
