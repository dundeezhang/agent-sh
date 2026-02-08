package tools

import (
	"fmt"
	"html"
	"io"
	"net"
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ToolResult{Content: fmt.Sprintf("search failed: HTTP %d %s", resp.StatusCode, resp.Status), IsError: true}
	}

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

func parseDDGResults(rawHTML string) []searchResult {
	links := reResult.FindAllStringSubmatch(rawHTML, -1)
	snippets := reSnippet.FindAllStringSubmatch(rawHTML, -1)

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

	// SSRF protection: reject private/loopback IPs.
	if err := checkSSRF(parsed.Hostname()); err != nil {
		return ToolResult{Content: err.Error(), IsError: true}
	}

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("error creating request: %s", err), IsError: true}
	}
	req.Header.Set("User-Agent", "agent-sh/1.0")

	// Use a client that re-checks SSRF on every redirect hop.
	safeClient := &http.Client{
		Timeout: httpClient.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			if err := checkSSRF(req.URL.Hostname()); err != nil {
				return err
			}
			return nil
		},
	}

	resp, err := safeClient.Do(req)
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

// checkSSRF rejects hostnames that resolve to private, loopback, or link-local IPs.
func checkSSRF(host string) error {
	if host == "localhost" {
		return fmt.Errorf("blocked: localhost is not allowed")
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		// If DNS fails, let the HTTP client deal with it.
		return nil
	}

	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return fmt.Errorf("blocked: %s resolves to private/loopback address %s", host, ip)
		}
	}
	return nil
}

// Pre-compiled regexps for htmlToText.
var (
	reScript = regexp.MustCompile(`(?si)<script[^>]*>.*?</script>`)
	reStyle  = regexp.MustCompile(`(?si)<style[^>]*>.*?</style>`)
	reBlock      = regexp.MustCompile(`(?i)</(p|div|h[1-6]|li|tr|br|hr)[^>]*>`)
	reBR         = regexp.MustCompile(`(?i)<br[^>]*/?>`)
	reSpaces     = regexp.MustCompile(`[^\S\n]+`)
	reBlankLines = regexp.MustCompile(`\n{3,}`)
)

// htmlToText strips HTML tags and decodes entities, collapsing whitespace.
func htmlToText(rawHTML string) string {
	rawHTML = reScript.ReplaceAllString(rawHTML, "")
	rawHTML = reStyle.ReplaceAllString(rawHTML, "")
	rawHTML = reBlock.ReplaceAllString(rawHTML, "\n")
	rawHTML = reBR.ReplaceAllString(rawHTML, "\n")

	text := stripTags(rawHTML)
	text = html.UnescapeString(text)

	text = reSpaces.ReplaceAllString(text, " ")
	text = reBlankLines.ReplaceAllString(text, "\n\n")

	return strings.TrimSpace(text)
}

func stripTags(s string) string {
	return reTag.ReplaceAllString(s, "")
}
