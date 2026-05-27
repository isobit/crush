package tools

import (
	"context"
	"fmt"

	kagi "github.com/kagisearch/kagi-openapi-golang"
)

// searchKagi performs a web search using the Kagi Search API.
func searchKagi(ctx context.Context, client *kagi.APIClient, query string, maxResults int) ([]SearchResult, error) {
	if maxResults <= 0 {
		maxResults = 10
	}

	req := kagi.NewSearchRequest(query)
	if maxResults > 0 {
		limit := int32(maxResults)
		req.Limit = &limit
	}

	resp, _, err := client.SearchAPI.Search(ctx).SearchRequest(*req).Execute()
	if err != nil {
		return nil, fmt.Errorf("kagi search failed: %w", err)
	}

	if resp == nil || resp.Data == nil {
		return nil, nil
	}

	var results []SearchResult
	for i, r := range resp.Data.Search {
		snippet := ""
		if r.Snippet != nil {
			snippet = *r.Snippet
		}
		results = append(results, SearchResult{
			Title:    r.Title,
			Link:     r.Url,
			Snippet:  snippet,
			Position: i + 1,
		})
		if len(results) >= maxResults {
			break
		}
	}
	return results, nil
}
