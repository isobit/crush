package tools

import (
	"context"
	_ "embed"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"charm.land/fantasy"
	kagi "github.com/kagisearch/kagi-openapi-golang"

	"github.com/charmbracelet/crush/internal/config"
)

//go:embed web_search.md.tpl
var webSearchDescriptionTmpl []byte

var webSearchDescriptionTpl = template.Must(
	template.New("webSearchDescription").
		Parse(string(webSearchDescriptionTmpl)),
)

//go:embed web_search_kagi.md.tpl
var kagiSearchDescriptionTmpl []byte

var kagiSearchDescriptionTpl = template.Must(
	template.New("kagiSearchDescription").
		Parse(string(kagiSearchDescriptionTmpl)),
)

// NewWebSearchTool creates a web search tool. The provider is selected based
// on cfg.Provider: "kagi" uses the Kagi Search API, anything else uses
// DuckDuckGo.
func NewWebSearchTool(client *http.Client, cfg config.ToolWebSearch, resolveKagiKey func() (string, error)) fantasy.AgentTool {
	if client == nil {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.MaxIdleConns = 100
		transport.MaxIdleConnsPerHost = 10
		transport.IdleConnTimeout = 90 * time.Second

		client = &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		}
	}

	if cfg.Provider == "kagi" {
		return newKagiSearchTool(resolveKagiKey)
	}
	return newDuckDuckGoSearchTool(client)
}

func newDuckDuckGoSearchTool(client *http.Client) fantasy.AgentTool {
	return fantasy.NewParallelAgentTool(
		WebSearchToolName,
		renderToolDescription(webSearchDescriptionTpl),
		func(ctx context.Context, params WebSearchParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.Query == "" {
				return fantasy.NewTextErrorResponse("query is required"), nil
			}

			maxResults := params.MaxResults
			if maxResults <= 0 {
				maxResults = 10
			}
			if maxResults > 20 {
				maxResults = 20
			}

			maybeDelaySearch()
			results, err := searchDuckDuckGo(ctx, client, params.Query, maxResults)
			slog.Debug("Web search completed", "query", params.Query, "results", len(results), "err", err)
			if err != nil {
				return fantasy.NewTextErrorResponse("Failed to search: " + err.Error()), nil
			}

			return fantasy.NewTextResponse(formatSearchResults(results)), nil
		},
	)
}

// newKagiSearchTool builds the Kagi tool. The key is resolved on each call
// rather than at construction, so config reloads and secret-manager lookups
// are honored without rebuilding the tool.
func newKagiSearchTool(resolveKagiKey func() (string, error)) fantasy.AgentTool {
	return fantasy.NewParallelAgentTool(
		WebSearchToolName,
		renderToolDescription(kagiSearchDescriptionTpl),
		func(ctx context.Context, params WebSearchParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.Query == "" {
				return fantasy.NewTextErrorResponse("query is required"), nil
			}

			maxResults := params.MaxResults
			if maxResults <= 0 {
				maxResults = 10
			}
			if maxResults > 20 {
				maxResults = 20
			}

			apiKey, err := resolveKagiKey()
			if err != nil {
				slog.Error("Failed to resolve Kagi API key", "error", err)
				return fantasy.NewTextErrorResponse("Failed to search: " + err.Error()), nil
			}
			results, err := searchKagi(ctx, newKagiClient(apiKey), params.Query, maxResults)
			if err != nil {
				slog.Error("Kagi search failed", "error", err)
				return fantasy.NewTextErrorResponse("Failed to search: " + err.Error()), nil
			}
			slog.Debug("Kagi search completed", "query", params.Query, "results", len(results))

			return fantasy.NewTextResponse(formatSearchResults(results)), nil
		},
	)
}

func newKagiClient(apiKey string) *kagi.APIClient {
	cfg := kagi.NewConfiguration()
	cfg.AddDefaultHeader("Authorization", "Bearer "+apiKey)
	return kagi.NewAPIClient(cfg)
}
