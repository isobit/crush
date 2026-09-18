package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

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

	resp, httpResp, err := client.SearchAPI.Search(ctx).SearchRequest(*req).Execute()
	if err != nil {
		return nil, formatKagiSearchError(httpResp, err)
	}

	if httpResp != nil && httpResp.StatusCode != http.StatusOK {
		return nil, formatKagiSearchError(httpResp, fmt.Errorf("unexpected HTTP status"))
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

const maxKagiErrorResponseBody = 8 * 1024

func formatKagiSearchError(resp *http.Response, err error) error {
	if resp == nil {
		return fmt.Errorf("Kagi search failed: %w", err)
	}

	status := resp.Status
	if trace := resp.Header.Get("X-Kagi-Trace"); trace != "" {
		status += "; trace: " + trace
	}

	if resp.Body == nil {
		return fmt.Errorf("Kagi search failed with HTTP status %s: %w", status, err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxKagiErrorResponseBody+1))
	if readErr != nil {
		return fmt.Errorf("Kagi search failed with HTTP status %s: %w (failed to read response body: %v)", status, err, readErr)
	}
	bodyText := strings.TrimSpace(string(body))
	if len(body) > maxKagiErrorResponseBody {
		bodyText = strings.TrimSpace(string(body[:maxKagiErrorResponseBody])) + "..."
	}
	if bodyText == "" {
		return fmt.Errorf("Kagi search failed with HTTP status %s: %w", status, err)
	}

	return fmt.Errorf("Kagi search failed with HTTP status %s: %w; response body: %s", status, err, bodyText)
}
