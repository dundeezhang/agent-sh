package tools

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/dundeezhang/agent-sh/internal/provider"
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

func (r *Registry) registerWebSearch() {
	r.register("web_search", provider.Tool{
		Name:        "web_search",
		Description: "Search the web using DuckDuckGo. Returns a list of result titles, URLs, and snippets. Use this when you need to look up documentation, find how to use a library or command, or research something you're unsure about.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "The search query",
				},
			},
			"required": []string{"query"},
		},
	}, executeWebSearch)
}

func (r *Registry) registerWebFetch() {
	r.register("web_fetch", provider.Tool{
		Name:        "web_fetch",
		Description: "Fetch a web page and return its text content (HTML tags stripped). Use this to read documentation pages, API references, or any URL found via web_search.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "The URL to fetch",
				},
			},
			"required": []string{"url"},
		},
	}, executeWebFetch)
}

func executeWebSearch(input map[string]interface{}) ToolResult {
	query, ok := input["query"].(string)
	if !ok || query == "" {
		return ToolResult{Content: "query is required", IsError: true}
	}

	u := "https://html.duckduckgo.com/html/?q=" + url.QueryEscape(query)
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("error creating request: %s", err), IsError: true}
	}
	req.Header.Set("User-Agent", "agent-sh/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("search error: %s", err), IsError: true}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("error reading response: %s", err), IsError: true}
	}

	results := parseDDGResults(string(body))
	if len(results) == 0 {
		return ToolResult{Content: "no results found"}
	}

	var sb strings.Builder
	for i, r := range results {
		if i >= 8 {
			break
		}
		fmt.Fprintf(&sb, "%d. %s\n   %s\n   %s\n\n", i+1, r.title, r.url, r.snippet)
	}
	return ToolResult{Content: sb.String()}
}

type searchResult struct {
	title   string
	url     string
	snippet string
}

var (
	reResult  = regexp.MustCompile(`(?s)<a[^>]*class="result__a"[^>]*href="([^"]*)"[^>]*>(.*?)</a>`)
	reSnippet = regexp.MustCompile(`(?s)<a[^>]*class="result__snippet"[^>]*>(.*?)</a>`)
	reTag     = regexp.MustCompile(`<[^>]*>`)
)

func parseDDGResults(html string) []searchResult {
	links := reResult.FindAllStringSubmatch(html, -1)
	snippets := reSnippet.FindAllStringSubmatch(html, -1)

	var results []searchResult
	for i, link := range links {
		if len(link) < 3 {
			continue
		}
		rawURL := link[1]
		title := stripTags(link[2])

		// DDG wraps URLs in a redirect; extract the actual URL.
		if parsed, err := url.Parse(rawURL); err == nil {
			if actual := parsed.Query().Get("uddg"); actual != "" {
				rawURL = actual
			}
		}

		snippet := ""
		if i < len(snippets) && len(snippets[i]) >= 2 {
			snippet = stripTags(snippets[i][1])
		}

		results = append(results, searchResult{
			title:   strings.TrimSpace(title),
			url:     rawURL,
			snippet: strings.TrimSpace(snippet),
		})
	}
	return results
}

func executeWebFetch(input map[string]interface{}) ToolResult {
	rawURL, ok := input["url"].(string)
	if !ok || rawURL == "" {
		return ToolResult{Content: "url is required", IsError: true}
	}

	// Basic URL validation.
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return ToolResult{Content: "invalid URL: must be http or https", IsError: true}
	}

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("error creating request: %s", err), IsError: true}
	}
	req.Header.Set("User-Agent", "agent-sh/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("fetch error: %s", err), IsError: true}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return ToolResult{Content: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status), IsError: true}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("error reading body: %s", err), IsError: true}
	}

	text := htmlToText(string(body))

	// Limit output size for the LLM context.
	runes := []rune(text)
	if len(runes) > 20000 {
		text = string(runes[:20000]) + "\n\n... [content truncated]"
	}

	if strings.TrimSpace(text) == "" {
		return ToolResult{Content: "(page returned no readable text)"}
	}

	return ToolResult{Content: text}
}

// htmlToText strips HTML tags and decodes common entities, collapsing whitespace.
func htmlToText(html string) string {
	// Remove script and style blocks.
	reScript := regexp.MustCompile(`(?si)<(script|style)[^>]*>.*?</\1>`)
	html = reScript.ReplaceAllString(html, "")

	// Replace block-level tags with newlines.
	reBlock := regexp.MustCompile(`(?i)</(p|div|h[1-6]|li|tr|br|hr)[^>]*>`)
	html = reBlock.ReplaceAllString(html, "\n")
	reBR := regexp.MustCompile(`(?i)<br[^>]*/?>`)
	html = reBR.ReplaceAllString(html, "\n")

	// Strip remaining tags.
	text := stripTags(html)

	// Decode common HTML entities.
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&nbsp;", " ")

	// Collapse whitespace.
	reSpaces := regexp.MustCompile(`[^\S\n]+`)
	text = reSpaces.ReplaceAllString(text, " ")
	reBlankLines := regexp.MustCompile(`\n{3,}`)
	text = reBlankLines.ReplaceAllString(text, "\n\n")

	return strings.TrimSpace(text)
}

func stripTags(s string) string {
	return reTag.ReplaceAllString(s, "")
}
