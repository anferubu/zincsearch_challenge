package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"enron-email-search/shared"
)

func QueryZincSearch(params SearchParams) (*SearchResponse, error) {
	config, err := shared.LoadConfig()

	if err != nil {
		return nil, err
	}

	query := buildQuery(params)
	queryJSON, err := json.Marshal(query)

	if err != nil {
		return nil, err
	}

	fmt.Printf("Payload enviado a ZincSearch: %s\n", string(queryJSON))

	searchURL := fmt.Sprintf("%s/es/%s/_search", config.ZINC_HOST, config.ZINC_INDEX)
	req, err := http.NewRequest("POST", searchURL, bytes.NewBuffer(queryJSON))

	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(config.ZINC_USER, config.ZINC_PASSWORD)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var zincResp ZincSearchResponse

	if err := json.NewDecoder(resp.Body).Decode(&zincResp); err != nil {
		return nil, err
	}

	emails := []shared.Email{}

	for _, hit := range zincResp.Hits.Hits {
		emails = append(emails, shared.Email{
			ID:       hit.Source.ID,
			Body:     hit.Source.Body,
			Datetime: hit.Source.Datetime,
			From:     hit.Source.From,
			To:       hit.Source.To,
			Subject:  hit.Source.Subject,
		})
	}

	totalPages := (zincResp.Hits.Total.Value + params.PageSize - 1) / params.PageSize

	return &SearchResponse{
		Emails:     emails,
		Total:      zincResp.Hits.Total.Value,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}

func buildQuery(params SearchParams) map[string]interface{} {
	var query map[string]interface{}

	if params.Query != "" || params.From != "" || params.To != "" || params.DateTime != "" {
		must := []map[string]interface{}{}
		should := []map[string]interface{}{}
		filter := []map[string]interface{}{}

		if params.Query != "" {
			should = append(should, map[string]interface{}{
				"match": map[string]interface{}{
					"body": params.Query,
				},
			})
			should = append(should, map[string]interface{}{
				"match": map[string]interface{}{
					"subject": params.Query,
				},
			})
		}

		if params.From != "" {
			must = append(must, map[string]interface{}{
				"term": map[string]interface{}{
					"from": params.From,
				},
			})
		}

		if params.To != "" {
			must = append(must, map[string]interface{}{
				"term": map[string]interface{}{
					"to": params.To,
				},
			})
		}

		if params.DateTime != "" {
			filter = append(filter, map[string]interface{}{
				"range": map[string]interface{}{
					"datetime": map[string]interface{}{
						"gte": params.DateTime + "T00:00:00.000Z",
						"lte": params.DateTime + "T23:59:59.999Z",
					},
				},
			})
		}

		query = map[string]interface{}{
			"query": map[string]interface{}{
				"bool": map[string]interface{}{
					"must":                 must,
					"should":               should,
					"minimum_should_match": 1,
				},
			},
			"from": (params.Page - 1) * params.PageSize,
			"size": params.PageSize,
		}

		// Add 'filter' only if it is not empty
		if len(filter) > 0 {
			query["query"].(map[string]interface{})["bool"].(map[string]interface{})["filter"] = filter
		}

	} else {
		query = map[string]interface{}{
			"query": map[string]interface{}{
				"match_all": map[string]interface{}{},
			},
			"from": (params.Page - 1) * params.PageSize,
			"size": params.PageSize,
		}
	}

	if params.SortBy != "" {
		query["sort_fields"] = []string{fmt.Sprintf("%s:%s", params.SortBy, params.SortDir)}
	}

	return query
}
