# asksh 구현 계획

## 컨텍스트

`asksh`는 자연어(한국어/영어)를 Ollama(로컬) 또는 OpenAI를 사용해 쉘 명령어로 번역하고, 2단계 안전 시스템과 대화형 확인을 제공하는 신규 Go CLI입니다. 전체 명세는 `SPEC.md`에 있습니다 (확정됨).

**목표:** 각 단계가 끝날 때마다 바이너리가 실행되고 새로운 기능을 보여줄 수 있도록 수직 슬라이스 방식으로 작업을 순서화합니다 — 수평적 레이어 방식 아님.

---

## 의존성 그래프

```
P1 스켈레톤 ─┬─► P2 설정 ──┬─► P3 LLM 번역 ─┬─► P4 안전 ─► P5 실행+확인 ─► P6 정리
             │             │                │
             └─► (shellctx) ┘               └─► (prompts/*.tmpl embed)
```

핵심 경로: **P1 → P3 → P5**

---

## Phase 1 — 스켈레톤 & Dry-Run 에코

이 단계 완료 후: `go build`로 `asksh` 생성; `asksh --dry-run "list files"`가 쿼리 + 쉘 컨텍스트를 출력, LLM 없음.

### Task 1.1 — 모듈 부트스트랩 + cobra 루트
- `go.mod` (Go 1.22+), 의존성: `spf13/cobra`, `BurntSushi/toml`, `fatih/color`, `golang.org/x/term`.
- `main.go` → `cmd.Execute()`.
- `cmd/root.go`: cobra 루트, 플래그 `--dry-run --ollama --openai --version`. 위치 인자를 하나의 쿼리로 합침.
- `cmd/config.go`: `config` 서브커맨드 등록 (스텁 본문).
- `--version`은 상수 `0.1.0-dev` 출력.

**인수 조건:** `--version`, `--help`, `--dry-run "x"` 모두 동작; bare `asksh`는 에러.
**검증:** `go build -o asksh ./...`; `cmd/root_test.go` 테이블 기반 테스트.

### Task 1.2 — ShellContext 수집기
- `internal/context/shell.go`: `type ShellContext struct { CWD, OS, Shell, Lang string }` + `Collect()`.
- `CWD`는 `os.Getwd`; `OS`는 `sw_vers -productVersion`, `runtime.GOOS` 폴백; `Shell`은 `$SHELL` basename. `Lang`은 P3까지 비워둠.
- 테스트가 쉘 아웃하지 않도록 `sw_vers` 주입 가능하게 만들기.
- `--dry-run`이 수집된 컨텍스트 출력.

**인수 조건:** `asksh --dry-run "x"` 실행 시 `cwd=… os=… shell=…` 표시.

> **CHECKPOINT A** — LLM 코드 작성 전에 CLI 플래그 이름, dry-run 출력 형식 확인.

---

## Phase 2 — 설정 로드 + 마법사

이 단계 완료 후: `asksh config`가 `~/.config/asksh/config.toml` 작성 (모드 0600); `--dry-run`이 해석된 백엔드 출력.

### Task 2.1 — 설정 스키마 + 로드/저장
- `internal/config/config.go`: SPEC §2.5를 반영하는 구조체. `Load`, `Save`, `DefaultConfig`, `ConfigPath` (XDG 인식).
- `Save`는 `MkdirAll(dir, 0700)` + `WriteFile(path, data, 0600)`; 쓰기 후 모드 검증.
- `OPENAI_API_KEY` 환경 변수 폴백은 로드 시가 아닌 사용 시 해석.

**인수 조건:** 라운드트립 테스트 `Save→Load→동일성 검증`; 파일 모드 `0600`.

### Task 2.2 — `asksh config` 마법사
- `internal/config/wizard.go`: `bufio.Scanner` 기반 프롬프트.
- 질문 항목: 백엔드, Ollama 호스트/모델, OpenAI 모델, OpenAI 키 (`term.ReadPassword`), `extra_llm_check`, `require_confirmation`. **결정 사항 #3에 따라 `[history]` 생략**.
- 기본값 미리 채우기; 빈 입력은 기본값 유지.

**인수 조건:** API 키 프롬프트에서 입력이 화면에 표시되지 않음; 파일 모드 `-rw-------`.

### Task 2.3 — 백엔드 재정의 플래그가 설정 사용
- `cmd/root.go`에서 유효 백엔드 해석: 플래그 우선; 없으면 `config.Backend.Default`.
- `Config`를 파라미터로 전달 — 전역 상태 없음.

**인수 조건:** 설정 기본값이 ollama일 때 `asksh --openai --dry-run "x"` 실행 시 `backend=openai` 출력.

> **CHECKPOINT B** — LLM 코드 확정 전에 TOML 키/기본값이 SPEC §2.5와 정확히 일치하는지 검증.

---

## Phase 3 — 번역

이 단계 완료 후: `asksh "list .log files"`가 실제 번역된 명령(청록색) 출력, 실행 없음. 한국어 입력 라운드트립.

### Task 3.1 — LLM 클라이언트 인터페이스 + 프롬프트 템플릿
- `internal/llm/client.go`: `Translate`, `SafetyCheck`, `Explain`을 가진 `Client` 인터페이스. `Verdict` ∈ `safe|warn|dangerous`.
- `prompts/translate.tmpl`과 `prompts/safety.tmpl`. `//go:embed prompts/*.tmpl`로 임베드.
- 번역 프롬프트: "쉘 명령만 반환하세요. 산문, 코드 펜스, 앞의 $ 없이."
- 언어 감지: 한글 범위 스캔 → `Lang = "ko"|"en"`. 단일 정규식, 단위 테스트 가능.

### Task 3.2 — Ollama HTTP 클라이언트
- `internal/llm/ollama.go`: `{host}/api/chat`에 POST, `"stream": false`, stdlib HTTP. 60초 타임아웃.
- 방어적 정리: 코드 펜스, `$ `, 후행 개행 제거.

**검증:** `httptest.Server` 단위 테스트. CI에서 실제 Ollama 없음.

### Task 3.3 — OpenAI HTTP 클라이언트
- `internal/llm/openai.go`: `https://api.openai.com/v1/chat/completions`에 POST. 설정 후 환경 변수에서 키 해석.

**검증:** `httptest.Server` 단위 테스트.

### Task 3.4 — 루트 명령에 번역 연결
- 설정+플래그로 클라이언트 빌드, `Translate` 호출, 청록색으로 명령 출력.
- `--dry-run`: 출력 후 exit 0. 그 외: "[실행 미구현]" 플레이스홀더.

**인수 조건:** Ollama에 한국어 쿼리 → 쉘 명령 출력; `--dry-run`은 절대 실행 안 함.

> **CHECKPOINT C** — `prompts/translate.tmpl` 반복 수정. 한국어 10개 + 영어 10개 쿼리 실행; 명령만 반환하는 계약이 지켜질 때까지 조정. P4 전에 프롬프트 확정.

---

## Phase 4 — 안전: 규칙 + LLM 검사

이 단계 완료 후: 위험 패턴 플래그 처리; 차단 패턴 즉시 중단. `--dry-run`에 판정 표시.

### Task 4.1 — 1단계 규칙 기반 차단 목록
- `internal/safety/rules.go`: `Verdict` ∈ `Safe|Warn|Dangerous|Blocked`; `func Check(cmd string) (Verdict, reason)`.
- DANGEROUS: `rm -rf`, `rm -r`, `dd`, `mkfs`, `fdisk`, `diskutil erase`, `chmod -R 777`, `chown -R`, `kill -9`, `> /dev/`, `truncate`, `shred`, `shutdown`, `reboot`, `git push --force`, `git reset --hard`, `git clean -fd`, 모든 `sudo`.
- BLOCKED: `curl|sh`, `wget|bash`, `sudo rm -rf /`, `/` 또는 `~`를 대상으로 하는 rm.
- `|`, `;`, `&&`로 분할 후 토큰화하여 헤드 매칭.

**검증:** 모든 패턴 + 거짓 양성 가드를 포함하는 테이블 기반 테스트.

### Task 4.2 — 2단계 비동기 LLM 안전성 검사
- `internal/safety/llmcheck.go`: `Probe(ctx, c, cmd) <-chan ProbeResult`. 고루틴이 `{Verdict, Reason, Err}` 반환.
- `prompts/safety.tmpl`: 정확히 `safe|warn|dangerous` + 한 줄 이유 응답.

**검증:** 각 판정에 목 `Client`; `ctx` 취소 테스트.

### Task 4.3 — 판정 결합 + 표시
- 최대 심각도 우선: `Blocked` > `Dangerous` > `Warn` > `Safe`.
- `--dry-run`에 두 판정 모두 출력.

**인수 조건:**
- `--dry-run "rm -rf /"` → `BLOCKED`, 비정상 종료.
- `--dry-run "rm -rf ./node_modules"` → `DANGEROUS (rule)` + `LLM: dangerous`.
- `--dry-run "ls"` → `SAFE`.

> **CHECKPOINT D** — exec 가능한 코드 작성 전에 차단 목록 패턴 + 결합 로직 감사.

---

## Phase 5 — 확인 프롬프트 + 실행기

이 단계 완료 후: 완전한 end-to-end. 안전 명령은 실행; 위험 명령은 `[y/n/e/?]` UI 표시.

### Task 5.1 — 확인 프롬프트 UI
- `internal/prompt/confirm.go`: `Ask(p Prompt) (Action, editedCmd string, err error)`. `Action ∈ {Execute, Cancel, Edit, Explain}`.
- SPEC §2.4 레이아웃 렌더링. 경고 빨간색, 명령 청록색, 옵션 흐리게.
- `e`: 재입력 요청; 수정된 명령은 1단계만 재실행 (2단계 아님).
- `?`: `Client.Explain` 호출; 출력 후 루프 반복.

**검증:** 주입된 `io.Reader` + `bytes.Buffer`; `y`, `n`, `e+edit+y`, `?+y` 드라이브.

### Task 5.2 — 실행기
- `internal/executor/run.go`: `Run(ctx, cmd) (exit int, err error)`, `exec.CommandContext(ctx, "sh", "-c", cmd)` 사용.
- stdout/stderr/stdin 상속; 종료 코드 전파.
- 심층 방어: `Run` 내부에서 `safety.Check(cmd)` 재확인; `Blocked`는 절대 exec 안 함.

**검증:** `echo hello` 테스트; `sudo rm -rf /`는 exec 없이 에러 반환.

### Task 5.3 — 루트 명령에 모든 단계 연결
최종 흐름:
1. 설정 로드; ShellContext 수집.
2. LLM 클라이언트 빌드.
3. `Translate` → 명령.
4. 명령 출력 (항상).
5. 1단계 `Check`. `Blocked`면 → exit 2.
6. 2단계 고루틴 시작.
7. Safe + 확인 불필요 → 즉시 실행. 그 외 2단계 대기 (≤ 2초) + 결합.
8. 판정 ≥ Warn → 확인 UI. 수정 루프는 1단계만.
9. 실행 → `executor.Run`.
10. `[history].enable`이면 → 로그 추가.

**인수 조건 — SPEC §2.1 예제 end-to-end.**

> **CHECKPOINT E** — 기능 완성. 30분 사용성 세션.

---

## Phase 6 — 정리, 히스토리, 배포

### Task 6.1 — 히스토리 로그 (선택 활성화)
- `internal/history/log.go`: `Append(path, entry)`. RFC3339 타임스탬프, 쿼리, 명령, 판정, 결과. 모드 0600. `config.History.Enable = true`일 때만.

### Task 6.2 — 에러, 종료 코드, 도움말 정리
- 종료 코드: 0 성공, 1 취소, 2 차단, 3 LLM 에러, 4 설정 에러.
- `NO_COLOR` + 비 tty 감지로 색상 비활성화.

### Task 6.3 — 배포
- darwin/arm64 + darwin/amd64용 `goreleaser.yml`.
- 설치, 예제, 문제 해결이 포함된 `README.md`.
- Homebrew 포뮬러 스텁.

> **CHECKPOINT F** — `v0.1.0` 태그, 클린 머신에서 스모크 테스트, 배포.

---

## 위험 및 열린 결정

1. Ollama 스트리밍: v1에서 `"stream": false` 사용; 지연 체감을 위한 스피너 추가.
2. 2단계 경쟁: 2단계를 ≤ 2초 대기 후 경고-건너뛰기.
3. 프롬프트 템플릿 조정: 가장 높은 레버리지; Checkpoint C에서 반복.
4. 차단 목록 거짓 양성: 토큰화된 매칭; `$HOME` 확장 결정 필요.
5. 설명: 번역 재사용이 아닌 전용 `Client.Explain` 메서드.
6. 수정 + 안전: 수정된 명령에는 1단계만 적용.
7. Cobra `config` 서브커맨드: "config"로 시작하는 쿼리는 따옴표 필요.
8. macOS 가드: `runtime.GOOS == "darwin"` 조건 하에 `sw_vers` 사용.

## v1 범위 외

- 별칭 제안, 스트리밍, 쉘 자동완성, Linux/Windows, CI에서 실제 LLM.
- 마법사에서 히스토리 (설정에서 수동으로 활성화 필요).

## 검증 (Phase 5 완료 후 end-to-end)

```
go test ./...
./asksh config
./asksh --dry-run "ls .log files"
./asksh "make a temp dir then list it"   # y 입력 후 실행
./asksh "rm -rf /"                       # BLOCKED, exit 2
./asksh "현재 브랜치의 마지막 5개 커밋 보기"  # git log -n 5
```
