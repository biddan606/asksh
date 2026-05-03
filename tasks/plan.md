# asksh Implementation Plan

## Context

`asksh` is a greenfield Go CLI that translates natural language (Korean/English) into shell commands using Ollama (local) or OpenAI, with a two-stage safety system and interactive confirmation. The full spec lives at `SPEC.md` (already finalized).

**Goal:** Order work in vertical slices so that after each phase the binary runs and demonstrates one new capability — not horizontal layers.

---

## Dependency Graph

```
P1 skeleton ─┬─► P2 config ──┬─► P3 LLM translate ─┬─► P4 safety ─► P5 execute+confirm ─► P6 polish
             │               │                      │
             └─► (shellctx) ─┘                      └─► (prompts/*.tmpl embed)
```

Critical path: **P1 → P3 → P5**

---

## Phase 1 — Skeleton & Dry-Run Echo

After this phase: `go build` produces `asksh`; `asksh --dry-run "list files"` prints query + shell context, no LLM.

### Task 1.1 — Bootstrap module + cobra root
- `go.mod` (Go 1.22+) with deps: `spf13/cobra`, `BurntSushi/toml`, `fatih/color`, `golang.org/x/term`.
- `main.go` → `cmd.Execute()`.
- `cmd/root.go`: cobra root, flags `--dry-run --ollama --openai --version`. Joins positional args into one query.
- `cmd/config.go`: register `config` subcommand (stub body).
- `--version` prints constant `0.1.0-dev`.

**Acceptance:** `--version`, `--help`, `--dry-run "x"` all work; bare `asksh` errors out.
**Verify:** `go build -o asksh ./...`; `cmd/root_test.go` table-driven tests.

### Task 1.2 — ShellContext collector
- `internal/context/shell.go`: `type ShellContext struct { CWD, OS, Shell, Lang string }` + `Collect()`.
- `CWD` from `os.Getwd`; `OS` via `sw_vers -productVersion` with `runtime.GOOS` fallback; `Shell` from `$SHELL` basename. `Lang` left empty until P3.
- Make `sw_vers` injectable so tests don't shell out.
- `--dry-run` prints the collected context.

**Acceptance:** `asksh --dry-run "x"` shows `cwd=… os=… shell=…`.

> **CHECKPOINT A** — Confirm CLI flag names, dry-run output format before any LLM code lands.

---

## Phase 2 — Config Load + Wizard

After this phase: `asksh config` writes `~/.config/asksh/config.toml` (mode 0600); `--dry-run` prints the resolved backend.

### Task 2.1 — Config schema + load/save
- `internal/config/config.go`: structs mirroring SPEC §2.5. `Load`, `Save`, `DefaultConfig`, `ConfigPath` (XDG-aware).
- `Save` does `MkdirAll(dir, 0700)` + `WriteFile(path, data, 0600)`; verify mode after write.
- `OPENAI_API_KEY` env-var fallback resolved at use time, not load time.

**Acceptance:** round-trip test `Save→Load→equal`; file mode is `0600`.

### Task 2.2 — `asksh config` wizard
- `internal/config/wizard.go`: `bufio.Scanner`-driven prompts.
- Questions: backend, Ollama host/model, OpenAI model, OpenAI key (`term.ReadPassword`), `extra_llm_check`, `require_confirmation`. **Skip `[history]`** per Decisions Log #3.
- Pre-fill defaults; empty input keeps default.

**Acceptance:** API key prompt doesn't echo; file mode `-rw-------`.

### Task 2.3 — Backend override flags consume config
- `cmd/root.go` resolves effective backend: flag wins; else `config.Backend.Default`.
- Plumb `Config` as parameter — no global state.

**Acceptance:** `asksh --openai --dry-run "x"` prints `backend=openai` when config default is ollama.

> **CHECKPOINT B** — Verify TOML keys/defaults match SPEC §2.5 exactly before LLM code locks them in.

---

## Phase 3 — Translation

After this phase: `asksh "list .log files"` prints a real translated command (cyan), no execution. Korean input round-trips.

### Task 3.1 — LLM client interface + prompt templates
- `internal/llm/client.go`: `Client` interface with `Translate`, `SafetyCheck`, `Explain`. `Verdict` ∈ `safe|warn|dangerous`.
- `prompts/translate.tmpl` and `prompts/safety.tmpl`. Embed via `//go:embed prompts/*.tmpl`.
- Translate prompt: "Return ONLY the shell command. No prose, no fences, no leading $."
- Language detection: Hangul-range scan → `Lang = "ko"|"en"`. Single regex, unit-testable.

### Task 3.2 — Ollama HTTP client
- `internal/llm/ollama.go`: POST `{host}/api/chat`, `"stream": false`, stdlib HTTP. 60s timeout.
- Defensive cleaning: strip code fences, `$ `, trailing newlines.

**Verify:** `httptest.Server` unit test. No real Ollama in CI.

### Task 3.3 — OpenAI HTTP client
- `internal/llm/openai.go`: POST `https://api.openai.com/v1/chat/completions`. Resolve key from config then env.

**Verify:** `httptest.Server` unit test.

### Task 3.4 — Wire translation into root command
- Build client from config+flags, call `Translate`, print command in cyan.
- `--dry-run`: print and exit 0. Otherwise: placeholder "[execution not yet implemented]".

**Acceptance:** Korean query against Ollama prints a shell command; `--dry-run` never executes.

> **CHECKPOINT C** — Iterate on `prompts/translate.tmpl`. Run 10 Korean + 10 English queries; tune until bare-command-only contract holds. Lock prompt before P4.

---

## Phase 4 — Safety: Rules + LLM Probe

After this phase: dangerous patterns flagged; blocked patterns hard-stop. Verdict visible in `--dry-run`.

### Task 4.1 — Stage 1 rule-based blocklist
- `internal/safety/rules.go`: `Verdict` ∈ `Safe|Warn|Dangerous|Blocked`; `func Check(cmd string) (Verdict, reason)`.
- DANGEROUS: `rm -rf`, `rm -r`, `dd`, `mkfs`, `fdisk`, `diskutil erase`, `chmod -R 777`, `chown -R`, `kill -9`, `> /dev/`, `truncate`, `shred`, `shutdown`, `reboot`, `git push --force`, `git reset --hard`, `git clean -fd`, any `sudo`.
- BLOCKED: `curl|sh`, `wget|bash`, `sudo rm -rf /`, rm targeting `/` or `~`.
- Tokenize + match heads after splitting on `|`, `;`, `&&`.

**Verify:** table-driven tests over every pattern + false-positive guards.

### Task 4.2 — Stage 2 async LLM safety probe
- `internal/safety/llmcheck.go`: `Probe(ctx, c, cmd) <-chan ProbeResult`. Goroutine yields `{Verdict, Reason, Err}`.
- `prompts/safety.tmpl`: answer exactly `safe|warn|dangerous` + one-line reason.

**Verify:** mock `Client` for each verdict; test `ctx` cancellation.

### Task 4.3 — Combine + display verdicts
- Max severity wins: `Blocked` > `Dangerous` > `Warn` > `Safe`.
- `--dry-run` prints both verdicts.

**Acceptance:**
- `--dry-run "rm -rf /"` → `BLOCKED`, exits non-zero.
- `--dry-run "rm -rf ./node_modules"` → `DANGEROUS (rule)` + `LLM: dangerous`.
- `--dry-run "ls"` → `SAFE`.

> **CHECKPOINT D** — Audit blocklist patterns + combine logic before any code can `exec`.

---

## Phase 5 — Confirm Prompt + Executor

After this phase: full end-to-end. Safe commands run; dangerous commands show `[y/n/e/?]` UI.

### Task 5.1 — Confirm prompt UI
- `internal/prompt/confirm.go`: `Ask(p Prompt) (Action, editedCmd string, err error)`. `Action ∈ {Execute, Cancel, Edit, Explain}`.
- Render SPEC §2.4 layout. Warning red, command cyan, options dim.
- `e`: re-prompt; edited command re-runs Stage 1 only (not Stage 2).
- `?`: call `Client.Explain`; print; loop back.

**Verify:** injected `io.Reader` + `bytes.Buffer`; drive `y`, `n`, `e+edit+y`, `?+y`.

### Task 5.2 — Executor
- `internal/executor/run.go`: `Run(ctx, cmd) (exit int, err error)` via `exec.CommandContext(ctx, "sh", "-c", cmd)`.
- Inherit stdout/stderr/stdin; propagate exit code.
- Defense in depth: re-check `safety.Check(cmd)` inside `Run`; never exec `Blocked`.

**Verify:** `echo hello` test; `sudo rm -rf /` returns error without exec.

### Task 5.3 — Wire all phases together in root command
Final flow:
1. Load config; collect ShellContext.
2. Build LLM client.
3. `Translate` → command.
4. Print command (always).
5. Stage 1 `Check`. If `Blocked` → exit 2.
6. Kick off Stage 2 goroutine.
7. If Safe + no confirmations required → execute immediately. Else wait Stage 2 (≤ 2s) + combine.
8. If verdict ≥ Warn → confirm UI. Edit loops with Stage 1 only.
9. Execute → `executor.Run`.
10. If `[history].enable` → append log.

**Acceptance — SPEC §2.1 examples end-to-end.**

> **CHECKPOINT E** — Feature complete. 30-min usability session.

---

## Phase 6 — Polish, History, Distribution

### Task 6.1 — History log (opt-in)
- `internal/history/log.go`: `Append(path, entry)`. RFC3339 timestamp, query, command, verdict, outcome. Mode 0600. Only when `config.History.Enable = true`. Not in wizard.

### Task 6.2 — Errors, exit codes, help polish
- Exit codes: 0 success, 1 cancel, 2 blocked, 3 LLM error, 4 config error.
- `NO_COLOR` + non-tty detection for color disable.

### Task 6.3 — Distribution
- `goreleaser.yml` for darwin/arm64 + darwin/amd64.
- `README.md` with install, examples, troubleshooting.
- Homebrew formula stub.

> **CHECKPOINT F** — Tag `v0.1.0`, smoke test on clean machine, ship.

---

## Risks & Open Decisions

1. Streaming Ollama: use `"stream": false` in v1; add spinner for perceived latency.
2. Stage-2 race: wait ≤ 2s for Stage 2, then warn-and-skip.
3. Prompt template tuning: highest leverage; iterate at Checkpoint C.
4. Blocklist false positives: tokenized matching; decide on `$HOME` expansion.
5. Explain: dedicated `Client.Explain` method, not reusing Translate.
6. Edit + safety: Stage 1 only on re-edited command.
7. Cobra `config` subcommand: any query starting with "config" must be quoted.
8. macOS guard: `sw_vers` behind `runtime.GOOS == "darwin"`.

## Out of Scope (v1)

- Alias suggestion, streaming, shell completion, Linux/Windows, real LLM in CI tests.
- History in wizard (must manually enable in config).

## Verification (end-to-end, after Phase 5)

```
go test ./...
./asksh config
./asksh --dry-run "ls .log files"
./asksh "make a temp dir then list it"   # runs after y
./asksh "rm -rf /"                       # BLOCKED, exit 2
./asksh "현재 브랜치의 마지막 5개 커밋 보기"  # git log -n 5
```
