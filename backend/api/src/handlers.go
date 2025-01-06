package api

import (
	"encoding/json"
	"net/http"
)

func SearchEmails(w http.ResponseWriter, r *http.Request) {
	page, pageSize := GetPaginationParams(r)

	params := SearchParams{
		Query:    r.URL.Query().Get("query"),
		From:     r.URL.Query().Get("from"),
		To:       r.URL.Query().Get("to"),
		DateTime: r.URL.Query().Get("dateTime"),
		SortBy:   r.URL.Query().Get("sortBy"),
		SortDir:  r.URL.Query().Get("sortDir"),
		Page:     page,
		PageSize: pageSize,
	}

	response, err := QueryZincSearch(params)

	if err != nil {
		http.Error(w, "Error searching for emails", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
