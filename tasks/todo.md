# asksh — 태스크 체크리스트

## Phase 1 — 스켈레톤 & Dry-Run 에코
- [x] **1.1** `go.mod` + cobra 루트 부트스트랩 (`main.go`, `cmd/root.go`, `cmd/config.go`)
- [x] **1.2** ShellContext 수집기 (`internal/context/shell.go`) + `--dry-run` 출력에 주입
- [ ] **CHECKPOINT A** — CLI 플래그 이름 & dry-run 출력 형식 검토

## Phase 2 — 설정 로드 + 마법사
- [ ] **2.1** 설정 스키마 + 로드/저장 (`internal/config/config.go`)
- [ ] **2.2** `asksh config` 대화형 마법사 (`internal/config/wizard.go`)
- [ ] **2.3** 백엔드 재정의 플래그가 설정 사용 (`cmd/root.go` 업데이트)
- [ ] **CHECKPOINT B** — TOML 키/기본값이 SPEC §2.5와 일치하는지 검증

## Phase 3 — 번역
- [ ] **3.1** LLM 클라이언트 인터페이스 + 프롬프트 템플릿 (`internal/llm/client.go`, `prompts/`)
- [ ] **3.2** Ollama HTTP 클라이언트 (`internal/llm/ollama.go`)
- [ ] **3.3** OpenAI HTTP 클라이언트 (`internal/llm/openai.go`)
- [ ] **3.4** 루트 명령에 번역 연결
- [ ] **CHECKPOINT C** — `prompts/translate.tmpl` 조정 (한국어 10개 + 영어 10개 쿼리)

## Phase 4 — 안전
- [ ] **4.1** 1단계 규칙 기반 차단 목록 (`internal/safety/rules.go`)
- [ ] **4.2** 2단계 비동기 LLM 안전성 검사 (`internal/safety/llmcheck.go`)
- [ ] **4.3** 판정 결합 + `--dry-run`에 표시
- [ ] **CHECKPOINT D** — 차단 목록 + 결합 로직 감사

## Phase 5 — 확인 프롬프트 + 실행기
- [ ] **5.1** 대화형 확인 프롬프트 UI (`internal/prompt/confirm.go`)
- [ ] **5.2** 쉘 명령 실행기 (`internal/executor/run.go`)
- [ ] **5.3** 루트 명령에 모든 단계 연결
- [ ] **CHECKPOINT E** — End-to-end 사용성 세션

## Phase 6 — 정리, 히스토리, 배포
- [ ] **6.1** 선택 활성화 히스토리 로그 (`internal/history/log.go`)
- [ ] **6.2** 종료 코드, `NO_COLOR`, 도움말 텍스트 정리
- [ ] **6.3** `goreleaser.yml` + `README.md` + Homebrew 포뮬러 스텁
- [ ] **CHECKPOINT F** — `v0.1.0` 태그, 스모크 테스트, 배포
