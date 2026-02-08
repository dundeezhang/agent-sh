# agent-sh

[![CodeFactor](https://www.codefactor.io/repository/github/dundeezhang/agent-sh/badge)](https://www.codefactor.io/repository/github/dundeezhang/agent-sh)
![CodeRabbit Pull Request Reviews](https://img.shields.io/coderabbit/prs/github/dundeezhang/agent-sh?utm_source=oss&utm_medium=github&utm_campaign=dundeezhang%2Fagent-sh&labelColor=171717&color=FF570A&link=https%3A%2F%2Fcoderabbit.ai&label=CodeRabbit+Reviews)

An AI-powered terminal shell. Use your shell as usual, and prefix any command with `@` to invoke an AI agent that can execute commands, read and write files, search code, and more. Natural language is automatically detected, so you can often skip the `@` prefix entirely.

Supports **Anthropic Claude**, **OpenAI**, and **Ollama** as LLM backends.

## Installation

Requires Go 1.25+.

```bash
# Clone and build
git clone https://github.com/dundeezhang/agent-sh.git
cd agent-sh
make build        # outputs to bin/agent-sh

# Or install directly to $GOPATH/bin
make install
```

## Quick Start

1. Set your API key:

```bash
export ANTHROPIC_API_KEY="sk-..."
```

1. Run the shell:

```bash
agent-sh
```

1. Use it like a normal shell, and prefix with `@` to talk to the AI agent:

```text
agent-sh ~/project> ls
agent-sh ~/project> git status
agent-sh ~/project> @ find all TODO comments in the codebase
agent-sh ~/project> @ refactor main.go to split the handler into separate functions
```

## Smart Command Detection

agent-sh automatically classifies your input as either a shell command or a natural language query using heuristic analysis. You often don't need the `@` prefix at all:

```text
agent-sh ~/project> find all TODO comments       # detected as natural language → agent
agent-sh ~/project> ls -la                        # detected as shell command → executed
agent-sh ~/project> what does main.go do          # detected as natural language → agent
```

If a command is not found (exit code 127), agent-sh automatically retries it as an AI query. Use `@@` to force a literal command starting with `@`.

## Agent Tools

The AI agent has access to the following tools:

| Tool | Description |
|------|-------------|
| `bash` | Execute shell commands (requires user confirmation) |
| `read_file` | Read file contents with optional offset and line limit |
| `write_file` | Create or overwrite files |
| `edit_file` | Find-and-replace editing within files |
| `search` | Regex search across files (uses ripgrep with grep fallback) |
| `glob` | Find files by glob pattern |
| `web_search` | Search the web via DuckDuckGo for docs, examples, and references |
| `web_fetch` | Fetch a URL and extract readable text content |

The agent runs in a loop of up to 25 turns, deciding which tools to call until the task is complete.

### Command Safety

Bash commands executed by the agent go through a safety check:

- **Read-only commands** (`ls`, `cat`, `git status`, `git log`, etc.) are **auto-approved** and run without prompting.
- **Mutating commands** (installations, file writes, destructive operations) require **explicit confirmation**.
- At the confirmation prompt, press **Enter** (yes), **n** (no), or **a** (auto-approve all commands for the session).
- Pipelines are only auto-approved if every command in the pipeline is read-only.
- Web fetch includes SSRF protection, blocking requests to localhost and private IPs.

## Markdown Rendering

Agent responses are rendered with ANSI styling in the terminal: **bold**, *italic*, `inline code`, and fenced code blocks are all styled for readability. A loading spinner with token usage stats is shown while the agent is thinking.

## Per-Directory Memory

agent-sh remembers the previous interaction in each directory. When you invoke the agent, it automatically receives context about what you did last time in that directory, enabling multi-step workflows across separate invocations.

Memory is stored in `~/.cache/agent-sh/context/` and is consumed after being read.

## Configuration

Configuration is loaded from `~/.config/agent-sh/config.toml`:

```toml
[api]
provider = "anthropic"   # anthropic, openai, or ollama
model = "sonnet"          # model name or alias (see below)
key = ""                  # API key (or use environment variables)
base_url = ""             # custom endpoint (used for Ollama)

[shell]
command = "/bin/zsh"      # underlying shell for command execution
prefix = "@"              # prefix to trigger the agent

[context]
history_size = 20         # number of recent commands included in agent context
include_git = true        # include git branch and status in agent context
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `ANTHROPIC_API_KEY` | Anthropic API key |
| `OPENAI_API_KEY` | OpenAI API key |
| `AGENT_SH_PROVIDER` | Override provider |
| `AGENT_SH_MODEL` | Override model |

### Model Aliases

**Anthropic:** `sonnet` (Claude Sonnet 4.5), `haiku` (Claude Haiku 4.5), `opus` (Claude Opus 4.6)

**OpenAI:** `gpt4` / `gpt-4` / `gpt4o` / `gpt-4o` (GPT-4o), `gpt4o-mini` / `gpt-4o-mini` (GPT-4o Mini)

**Ollama:** Any model name supported by your Ollama instance. Set provider to `ollama` and agent-sh will use `http://localhost:11434/v1` by default.

### CLI Flags

```bash
agent-sh -provider openai    # override provider
agent-sh -model gpt4o        # override model
agent-sh -version             # print version
```

## Shell Built-ins

| Command | Description |
|---------|-------------|
| `cd` | Change directory (supports `~`) |
| `exit` | Exit the shell |
| `export` | Set environment variables |
| `env` | List environment variables |
| `history` | Show command history |

Tab completion is supported for commands (from PATH and built-ins) and file paths. Multiple matches are displayed in columns, and directories get a trailing slash.

Interactive programs like `vim`, `less`, and `top` work correctly — agent-sh transfers terminal control to child processes and restores it when they exit.

## Building

```bash
make build     # build to bin/agent-sh
make install   # install to $GOPATH/bin
make test      # run tests
make clean     # remove build artifacts
```

## License

MIT License. See [LICENSE](LICENSE) for details.
