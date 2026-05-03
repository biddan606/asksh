# asksh Specification

## 1. Objective

**asksh** is a macOS CLI tool that translates natural language into shell commands and executes them locally. It targets developers who want to use CLI, git, and general shell operations without memorizing exact syntax.

**Target users:** macOS developers (beginner to intermediate) who know what they want to do but not always the exact command.

**Core value proposition:** Type intent in Korean or English → get the right shell command → run it safely.

---

## 2. Features & Acceptance Criteria

### 2.1 Core Translation

| Feature | Acceptance Criteria |
|---|---|
| Natural language → shell command | Given `asksh "현재 디렉토리의 모든 .log 파일 삭제"`, outputs `rm *.log` and prompts before execution |
| No domain restriction | Handles all shell commands: file ops, text processing, process mgmt, network, archives, brew, macOS-specific, git, and any other valid shell command |
| Language auto-detection | Detects Korean or English from input; responds and prompts in the same language |
| Context injection | Injects current working directory, OS version, and shell type into the LLM prompt for accurate translation |
| Shows command before execution | Always print the translated command before running it |

### 2.2 LLM Backends

| Backend | Details |
|---|---|
| Ollama (local) | Default. Connects to `http://localhost:11434`. Model configurable (default: `qwen2.5-coder:7b`) |
| OpenAI API | Requires API key in config. Model configurable (default: `gpt-4o-mini`) |
| Backend selection | Config sets default; `--ollama` / `--openai` flags override per-invocation |

### 2.3 Safety System

Two-stage check before execution:

**Stage 1 — Rule-based blocklist (synchronous, no LLM cost)**

Patterns that always require confirmation or are blocked:

```
DANGEROUS (require explicit confirmation):
  rm -rf, rm -r, dd, mkfs, fdisk, diskutil erase,
  chmod -R 777, chown -R, kill -9 (on broad patterns),
  > /dev/*, truncate, shred, shutdown, reboot,
  git push --force, git reset --hard, git clean -fd

BLOCKED (never execute, explain and exit):
  curl | sh, wget | bash  (remote code execution patterns)
  sudo rm -rf /
  Any command targeting / or ~ as rm target
```

**Stage 2 — LLM safety check (async, runs in parallel with stage 1 display)**

Ask the LLM: "Is this command potentially destructive or irreversible? Answer only: safe / warn / dangerous"

- `safe` → execute immediately after showing command
- `warn` → show warning and prompt user
- `dangerous` → block and explain

### 2.4 Interactive Confirmation Prompt

When a command is flagged (either stage):

```
Translated command: rm -rf ./node_modules

⚠  This command is destructive and irreversible.
   Reason: removes directory tree recursively

Options:
  [y] Execute anyway
  [n] Cancel
  [e] Edit command manually
  [?] Show explanation

Your choice [y/n/e/?]:
```

### 2.5 Configuration

Stored at `~/.config/asksh/config.toml`.

```toml
[backend]
default = "ollama"          # "ollama" | "openai"

[ollama]
host    = "http://localhost:11434"
model   = "qwen2.5-coder:7b"

[openai]
api_key = ""                # or set OPENAI_API_KEY env var
model   = "gpt-4o-mini"

[safety]
require_confirmation = true  # always show command before running
extra_llm_check      = true  # enable stage-2 LLM safety check

[history]
enable = false               # opt-in: log executed commands to ~/.config/asksh/history.log
```

**Setup command:** `asksh config` — interactive wizard that writes config.toml.

### 2.6 CLI Interface

```
asksh <natural language query>     # translate and run
asksh config                       # interactive config wizard
asksh --dry-run <query>            # translate only, do not execute
asksh --ollama <query>             # force Ollama backend
asksh --openai <query>             # force OpenAI backend
asksh --version
asksh --help
```

---

## 3. Project Structure

```
asksh/
├── main.go
├── cmd/
│   ├── root.go          # cobra root command, flag parsing
│   └── config.go        # `asksh config` wizard
├── internal/
│   ├── llm/
│   │   ├── client.go    # interface: Translate(query, shellctx) (string, error)
│   │   ├── ollama.go    # Ollama HTTP client
│   │   └── openai.go    # OpenAI REST client
│   ├── context/
│   │   └── shell.go     # collects cwd, os version, shell type → ShellContext struct
│   ├── safety/
│   │   ├── rules.go     # rule-based blocklist, returns: safe/warn/block + reason
│   │   └── llmcheck.go  # LLM safety probe
│   ├── executor/
│   │   └── run.go       # executes shell command via os/exec, streams output
│   ├── prompt/
│   │   └── confirm.go   # interactive y/n/e/? prompt (uses golang.org/x/term)
│   └── config/
│       ├── config.go    # load/save config.toml
│       └── wizard.go    # interactive setup wizard
│   └── history/
│       └── log.go       # append-only log writer, enabled by config
├── prompts/
│   ├── translate.tmpl   # system prompt: includes {{.CWD}}, {{.OS}}, {{.Shell}}, {{.Lang}}
│   └── safety.tmpl      # system prompt for safety check
├── go.mod
├── go.sum
├── SPEC.md
└── README.md
```

---

## 4. Code Style

- **Language:** Go 1.22+
- **CLI framework:** `github.com/spf13/cobra`
- **Config:** `github.com/BurntSushi/toml`
- **Terminal color/style:** `github.com/fatih/color`
- **No ORM, no frameworks** — stdlib HTTP for LLM API calls
- **Error handling:** wrap errors with `fmt.Errorf("context: %w", err)`, never silent
- **Prompt templates:** stored in `prompts/` as `.tmpl` files, embedded via `//go:embed`
- **No global state** — pass config explicitly through function parameters
- **Tests:** table-driven, in `_test.go` files alongside source

---

## 5. Testing Strategy

| Layer | Approach |
|---|---|
| Safety rules | Unit tests: table of (input command → expected verdict) covering all blocklist patterns |
| LLM clients | Interface mock; test request format and response parsing, not actual LLM output |
| Config | Round-trip: write config → read config → assert equality |
| Executor | Test with `echo` and `true` commands; test that dangerous commands are never passed to exec without confirmation |
| Integration | `--dry-run` mode end-to-end: input query → translated command printed, nothing executed |

**No tests that require a running Ollama or real OpenAI key in CI.** Use interface mocks.

---

## 6. Boundaries

### Always do
- Show the translated command to the user before executing
- Run stage-1 rule-based safety check on every command
- Respect `require_confirmation = true` in config
- Support `OPENAI_API_KEY` environment variable as fallback for the key
- Exit with non-zero status code on error

### Ask first (interactive confirmation required)
- Any command matching the DANGEROUS pattern list
- Any command flagged `warn` or `dangerous` by LLM safety check
- Commands with `sudo`

### Never do
- Execute a command without displaying it first
- Silently skip the safety check
- Store API keys anywhere other than `~/.config/asksh/config.toml` (file permissions: 0600)
- Execute patterns in the BLOCKED list under any circumstances
- Make network calls to any service other than the configured LLM backend

---

## 7. Distribution (macOS)

- **Primary:** Homebrew tap (`brew install biddan606/tap/asksh`)
- **Alternative:** Download pre-built binary from GitHub Releases
- **Build:** `go build -o asksh ./main.go` produces a single static binary

**Installation flow for users:**
```bash
brew install biddan606/tap/asksh
asksh config          # run setup wizard
asksh "현재 브랜치의 모든 커밋 로그 보기"
```

---

## Decisions Log

| # | Question | Decision |
|---|---|---|
| 1 | LLM prompt language | Auto-detect from input (Korean/English); prompt and UI respond in same language |
| 2 | Shell context injection | Yes — inject CWD, macOS version, shell type into every translation prompt |
| 3 | Command history log | Opt-in (`enable = false` by default); stored at `~/.config/asksh/history.log` with timestamps. Excluded from v1 wizard, user must manually enable. |
| 4 | Alias suggestion | Out of scope for v1. Deferred to avoid shell-config mutation complexity. |
