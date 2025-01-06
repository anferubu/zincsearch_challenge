package api

import "enron-email-search/shared"

// Query params for the endpoint /api/emails/
type SearchParams struct {
	Query    string `json:"query"`
	From     string `json:"from"`
	To       string `json:"to"`
	DateTime string `json:"dateTime"`
	SortBy   string `json:"sortBy"`
	SortDir  string `json:"sortDir"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
}

// Response's structure for the endpoint /api/emails/
type SearchResponse struct {
	Emails     []shared.Email `json:"emails"`
	Total      int            `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"pageSize"`
	TotalPages int            `json:"totalPages"`
}

// Body request for the ZincSearch endpoint /api/{index}/_search/
type ZincSearchRequest struct {
	SearchType string                 `json:"search_type"`
	Query      map[string]interface{} `json:"query"`
	SortFields []string               `json:"sort_fields,omitempty"`
	From       int                    `json:"from"`
	MaxResults int                    `json:"max_results"`
	Source     []string               `json:"_source,omitempty"`
}

// Response's structure for the ZincSearch endpoint /api/{index}/_search/
type ZincSearchResponse struct {
	Took     int     `json:"took"`
	TimedOut bool    `json:"timed_out"`
	MaxScore float64 `json:"max_score"`
	Error    string  `json:"error"`
	Hits     struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
		Hits []struct {
			Index     string  `json:"_index"`
			Type      string  `json:"_type"`
			ID        string  `json:"_id"`
			Score     float64 `json:"_score"`
			Timestamp string  `json:"@timestamp"`
			Source    struct {
				ID       string `json:"_id"`
				Body     string `json:"body"`
				Datetime string `json:"datetime"`
				From     string `json:"from"`
				To       string `json:"to"`
				Subject  string `json:"subject"`
			} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}
