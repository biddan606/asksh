# asksh — Task Checklist

## Phase 1 — Skeleton & Dry-Run Echo
- [x] **1.1** Bootstrap `go.mod` + cobra root (`main.go`, `cmd/root.go`, `cmd/config.go`)
- [ ] **1.2** ShellContext collector (`internal/context/shell.go`) + inject into `--dry-run` output
- [ ] **CHECKPOINT A** — Review CLI flag names & dry-run output format

## Phase 2 — Config Load + Wizard
- [ ] **2.1** Config schema + load/save (`internal/config/config.go`)
- [ ] **2.2** `asksh config` interactive wizard (`internal/config/wizard.go`)
- [ ] **2.3** Backend override flags consume config (update `cmd/root.go`)
- [ ] **CHECKPOINT B** — Verify TOML keys/defaults match SPEC §2.5

## Phase 3 — Translation
- [ ] **3.1** LLM client interface + prompt templates (`internal/llm/client.go`, `prompts/`)
- [ ] **3.2** Ollama HTTP client (`internal/llm/ollama.go`)
- [ ] **3.3** OpenAI HTTP client (`internal/llm/openai.go`)
- [ ] **3.4** Wire translation into root command
- [ ] **CHECKPOINT C** — Tune `prompts/translate.tmpl` (10 KO + 10 EN queries)

## Phase 4 — Safety
- [ ] **4.1** Stage 1 rule-based blocklist (`internal/safety/rules.go`)
- [ ] **4.2** Stage 2 async LLM safety probe (`internal/safety/llmcheck.go`)
- [ ] **4.3** Combine verdicts + display in `--dry-run`
- [ ] **CHECKPOINT D** — Audit blocklist + combine logic

## Phase 5 — Confirm Prompt + Executor
- [ ] **5.1** Interactive confirm prompt UI (`internal/prompt/confirm.go`)
- [ ] **5.2** Shell command executor (`internal/executor/run.go`)
- [ ] **5.3** Wire all phases together in root command
- [ ] **CHECKPOINT E** — End-to-end usability session

## Phase 6 — Polish, History, Distribution
- [ ] **6.1** Opt-in history log (`internal/history/log.go`)
- [ ] **6.2** Exit codes, `NO_COLOR`, help text polish
- [ ] **6.3** `goreleaser.yml` + `README.md` + Homebrew formula stub
- [ ] **CHECKPOINT F** — Tag `v0.1.0`, smoke test, ship
