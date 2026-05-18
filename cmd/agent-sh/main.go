package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/dundeezhang/agent-sh/internal/agent"
	"github.com/dundeezhang/agent-sh/internal/config"
	"github.com/dundeezhang/agent-sh/internal/memory"
	"github.com/dundeezhang/agent-sh/internal/provider"
	"github.com/dundeezhang/agent-sh/internal/render"
	"github.com/dundeezhang/agent-sh/internal/shell"
	"github.com/dundeezhang/agent-sh/internal/tools"
)

var version = "0.1.0"

func main() {
	modelFlag := flag.String("model", "", "Model to use (overrides config)")
	providerFlag := flag.String("provider", "", "Provider to use: anthropic, openai, ollama")
	versionFlag := flag.Bool("version", false, "Print version and exit")
	noBannerFlag := flag.Bool("no-banner", false, "Disable startup banner")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("agent-sh v%s\n", version)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %s\n", err)
		os.Exit(1)
	}

	// Flag overrides
	if *providerFlag != "" {
		cfg.API.Provider = *providerFlag
	}
	if *modelFlag != "" {
		cfg.API.Model = *modelFlag
	}

	// Initialize provider
	p, err := initProvider(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing provider: %s\n", err)
		os.Exit(1)
	}

	// Initialize tools and renderer
	registry := tools.NewRegistry()
	renderer := render.NewRenderer()

	// Create agent
	ag := agent.New(p, cfg.API.Model, registry, renderer, cfg.Context.IncludeGit)

	// Create history buffer
	history := shell.NewHistory(cfg.Context.HistorySize)

	// Build startup banner
	banner := fmt.Sprintf("agent-sh v%s | %s/%s", version, cfg.API.Provider, cfg.API.Model)
	showBanner := cfg.Shell.ShowBanner && !*noBannerFlag

	// Create and run shell
	sh := shell.New(history, func(input string) {
		cwd, _ := os.Getwd()

		// Read previous context for this directory, then delete it.
		var prevContext string
		if ctx, err := memory.Read(cwd); err != nil {
			fmt.Fprintf(os.Stderr, "warning: reading context: %s\n", err)
		} else if ctx != nil {
			prevContext = memory.Render(ctx)
			_ = memory.Delete(cwd)
		}

		result := ag.Run(input, history.Recent(0), prevContext)

		// Persist the new interaction context.
		if result != nil && cwd != "" {
			summaries := make([]memory.ToolCallSummary, len(result.ToolCalls))
			for i, tc := range result.ToolCalls {
				summaries[i] = memory.ToolCallSummary{
					Tool:    tc.Tool,
					Input:   tc.Input,
					IsError: tc.IsError,
				}
			}
			if err := memory.Write(cwd, &memory.Context{
				Timestamp: time.Now(),
				CWD:       cwd,
				Query:     result.Query,
				ToolCalls: summaries,
				Summary:   result.Summary,
			}); err != nil {
				fmt.Fprintf(os.Stderr, "warning: writing context: %s\n", err)
			}
		}
	}, shell.WithBanner(banner, showBanner))

	if err := sh.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Shell error: %s\n", err)
		os.Exit(1)
	}
}

const ollamaDefaultBaseURL = "http://localhost:11434/v1"

func initProvider(cfg *config.Config) (provider.Provider, error) {
	switch cfg.API.Provider {
	case "anthropic":
		if cfg.API.Key == "" && os.Getenv("ANTHROPIC_API_KEY") == "" {
			return nil, fmt.Errorf("anthropic API key required: set ANTHROPIC_API_KEY or configure in ~/.config/agent-sh/config.toml")
		}
		return provider.NewAnthropic(cfg.API.Key), nil
	case "openai":
		if cfg.API.Key == "" && os.Getenv("OPENAI_API_KEY") == "" {
			return nil, fmt.Errorf("OpenAI API key required: set OPENAI_API_KEY or configure in ~/.config/agent-sh/config.toml")
		}
		return provider.NewOpenAI(cfg.API.Key, cfg.API.BaseURL), nil
	case "ollama":
		baseURL := cfg.API.BaseURL
		if baseURL == "" {
			baseURL = ollamaDefaultBaseURL
		}
		return provider.NewOpenAI(cfg.API.Key, baseURL), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", cfg.API.Provider)
	}
}
