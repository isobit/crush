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
func NewWebSearchTool(client *http.Client, cfg config.ToolWebSearch) fantasy.AgentTool {
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
		return newKagiSearchTool(cfg.KagiAPIKey)
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

func newKagiSearchTool(apiKey string) fantasy.AgentTool {
	cfg := kagi.NewConfiguration()
	cfg.AddDefaultHeader("Authorization", "Bearer "+apiKey)
	kagiClient := kagi.NewAPIClient(cfg)

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

			results, err := searchKagi(ctx, kagiClient, params.Query, maxResults)
			slog.Debug("Kagi search completed", "query", params.Query, "results", len(results), "err", err)
			if err != nil {
				return fantasy.NewTextErrorResponse("Failed to search: " + err.Error()), nil
			}

			return fantasy.NewTextResponse(formatSearchResults(results)), nil
		},
	)
}
