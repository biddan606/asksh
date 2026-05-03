# asksh 명세서

## 1. 목표

**asksh**는 자연어를 쉘 명령어로 번역하고 로컬에서 실행하는 macOS CLI 도구입니다. 정확한 문법을 외우지 않고도 CLI, git, 일반 쉘 작업을 하고 싶은 개발자를 대상으로 합니다.

**대상 사용자:** 원하는 것은 알지만 정확한 명령어를 항상 기억하지 못하는 macOS 개발자 (초급~중급).

**핵심 가치:** 한국어나 영어로 의도를 입력 → 올바른 쉘 명령어 획득 → 안전하게 실행.

---

## 2. 기능 및 인수 기준

### 2.1 핵심 번역

| 기능 | 인수 기준 |
|---|---|
| 자연어 → 쉘 명령어 | `asksh "현재 디렉토리의 모든 .log 파일 삭제"` 입력 시 `rm *.log`를 출력하고 실행 전 확인 요청 |
| 도메인 제한 없음 | 모든 쉘 명령어 처리: 파일 조작, 텍스트 처리, 프로세스 관리, 네트워크, 아카이브, brew, macOS 전용, git 등 |
| 언어 자동 감지 | 입력에서 한국어 또는 영어를 감지하고 동일 언어로 응답 및 안내 |
| 컨텍스트 주입 | 정확한 번역을 위해 현재 작업 디렉토리, OS 버전, 쉘 종류를 LLM 프롬프트에 주입 |
| 실행 전 명령어 표시 | 번역된 명령어를 실행 전에 항상 출력 |

### 2.2 LLM 백엔드

| 백엔드 | 설명 |
|---|---|
| Ollama (로컬) | 기본값. `http://localhost:11434`에 연결. 모델 설정 가능 (기본값: `qwen2.5-coder:7b`) |
| OpenAI API | 설정에 API 키 필요. 모델 설정 가능 (기본값: `gpt-4o-mini`) |
| 백엔드 선택 | 설정에서 기본값 지정; `--ollama` / `--openai` 플래그로 호출 시 재정의 가능 |

### 2.3 안전 시스템

실행 전 2단계 검사:

**1단계 — 규칙 기반 차단 목록 (동기, LLM 비용 없음)**

항상 확인이 필요하거나 차단되는 패턴:

```
위험 (명시적 확인 필요):
  rm -rf, rm -r, dd, mkfs, fdisk, diskutil erase,
  chmod -R 777, chown -R, kill -9 (광범위 패턴),
  > /dev/*, truncate, shred, shutdown, reboot,
  git push --force, git reset --hard, git clean -fd

차단 (절대 실행 불가, 이유 설명 후 종료):
  curl | sh, wget | bash  (원격 코드 실행 패턴)
  sudo rm -rf /
  / 또는 ~ 를 대상으로 하는 rm 명령
```

**2단계 — LLM 안전성 검사 (비동기, 1단계 표시와 병렬 실행)**

LLM에 질문: "이 명령이 잠재적으로 파괴적이거나 되돌릴 수 없나요? safe / warn / dangerous 중 하나만 답하세요"

- `safe` → 명령 표시 후 즉시 실행
- `warn` → 경고 표시 후 사용자에게 확인 요청
- `dangerous` → 차단 후 이유 설명

### 2.4 대화형 확인 프롬프트

명령이 플래그 처리된 경우 (두 단계 모두):

```
번역된 명령어: rm -rf ./node_modules

⚠  이 명령은 파괴적이며 되돌릴 수 없습니다.
   이유: 디렉토리 트리를 재귀적으로 삭제합니다

선택:
  [y] 그래도 실행
  [n] 취소
  [e] 명령 직접 수정
  [?] 설명 보기

선택 [y/n/e/?]:
```

### 2.5 설정

`~/.config/asksh/config.toml`에 저장.

```toml
[backend]
default = "ollama"          # "ollama" | "openai"

[ollama]
host    = "http://localhost:11434"
model   = "qwen2.5-coder:7b"

[openai]
api_key = ""                # 또는 OPENAI_API_KEY 환경 변수 사용
model   = "gpt-4o-mini"

[safety]
require_confirmation = true  # 실행 전 항상 명령어 표시
extra_llm_check      = true  # 2단계 LLM 안전성 검사 활성화

[history]
enable = false               # 선택적 활성화: 실행된 명령을 ~/.config/asksh/history.log에 기록
```

**설정 명령:** `asksh config` — config.toml을 작성하는 대화형 마법사.

### 2.6 CLI 인터페이스

```
asksh <자연어 쿼리>            # 번역 및 실행
asksh config                  # 대화형 설정 마법사
asksh --dry-run <쿼리>        # 번역만, 실행하지 않음
asksh --ollama <쿼리>         # Ollama 백엔드 강제
asksh --openai <쿼리>         # OpenAI 백엔드 강제
asksh --version
asksh --help
```

---

## 3. 프로젝트 구조

```
asksh/
├── main.go
├── cmd/
│   ├── root.go          # cobra 루트 명령, 플래그 파싱
│   └── config.go        # `asksh config` 마법사
├── internal/
│   ├── llm/
│   │   ├── client.go    # 인터페이스: Translate(query, shellctx) (string, error)
│   │   ├── ollama.go    # Ollama HTTP 클라이언트
│   │   └── openai.go    # OpenAI REST 클라이언트
│   ├── context/
│   │   └── shell.go     # cwd, os 버전, 쉘 종류 수집 → ShellContext 구조체
│   ├── safety/
│   │   ├── rules.go     # 규칙 기반 차단 목록, 반환값: safe/warn/block + 이유
│   │   └── llmcheck.go  # LLM 안전성 검사
│   ├── executor/
│   │   └── run.go       # os/exec를 통한 쉘 명령 실행, 출력 스트리밍
│   ├── prompt/
│   │   └── confirm.go   # 대화형 y/n/e/? 프롬프트 (golang.org/x/term 사용)
│   └── config/
│       ├── config.go    # config.toml 로드/저장
│       └── wizard.go    # 대화형 설정 마법사
│   └── history/
│       └── log.go       # 추가 전용 로그 작성기, 설정에 의해 활성화
├── prompts/
│   ├── translate.tmpl   # 시스템 프롬프트: {{.CWD}}, {{.OS}}, {{.Shell}}, {{.Lang}} 포함
│   └── safety.tmpl      # 안전성 검사용 시스템 프롬프트
├── go.mod
├── go.sum
├── SPEC.md
└── README.md
```

---

## 4. 코드 스타일

- **언어:** Go 1.22+
- **CLI 프레임워크:** `github.com/spf13/cobra`
- **설정:** `github.com/BurntSushi/toml`
- **터미널 색상/스타일:** `github.com/fatih/color`
- **ORM 없음, 프레임워크 없음** — LLM API 호출에 stdlib HTTP 사용
- **에러 처리:** `fmt.Errorf("context: %w", err)`로 에러 래핑, 무시 금지
- **프롬프트 템플릿:** `prompts/`에 `.tmpl` 파일로 저장, `//go:embed`로 임베드
- **전역 상태 없음** — 설정을 함수 파라미터로 명시적으로 전달
- **테스트:** 테이블 기반, 소스 옆의 `_test.go` 파일에 작성

---

## 5. 테스트 전략

| 레이어 | 방식 |
|---|---|
| 안전 규칙 | 단위 테스트: 모든 차단 목록 패턴을 포함하는 (입력 명령 → 예상 판정) 테이블 |
| LLM 클라이언트 | 인터페이스 목(mock); 실제 LLM 출력이 아닌 요청 형식과 응답 파싱 테스트 |
| 설정 | 라운드트립: 설정 쓰기 → 읽기 → 동일성 검증 |
| 실행기 | `echo`, `true` 명령으로 테스트; 위험 명령이 확인 없이 exec에 전달되지 않는지 테스트 |
| 통합 | `--dry-run` 모드 end-to-end: 입력 쿼리 → 번역된 명령 출력, 아무것도 실행 안 됨 |

**CI에서 실행 중인 Ollama나 실제 OpenAI 키가 필요한 테스트 없음.** 인터페이스 목 사용.

---

## 6. 경계

### 항상 해야 할 것
- 실행 전 번역된 명령을 사용자에게 표시
- 모든 명령에 1단계 규칙 기반 안전성 검사 실행
- 설정의 `require_confirmation = true` 준수
- 키의 폴백으로 `OPENAI_API_KEY` 환경 변수 지원
- 에러 시 비정상 상태 코드로 종료

### 먼저 물어봐야 할 것 (대화형 확인 필요)
- DANGEROUS 패턴 목록에 일치하는 명령
- LLM 안전성 검사에서 `warn` 또는 `dangerous`로 플래그 처리된 명령
- `sudo`가 포함된 명령

### 절대 하지 말 것
- 먼저 표시하지 않고 명령 실행
- 안전성 검사를 조용히 건너뛰기
- `~/.config/asksh/config.toml` (파일 권한: 0600) 이외의 위치에 API 키 저장
- 어떤 상황에서도 BLOCKED 목록의 패턴 실행
- 설정된 LLM 백엔드 이외의 서비스에 네트워크 호출

---

## 7. 배포 (macOS)

- **주요:** Homebrew 탭 (`brew install biddan606/tap/asksh`)
- **대안:** GitHub Releases에서 사전 빌드된 바이너리 다운로드
- **빌드:** `go build -o asksh ./main.go`로 단일 정적 바이너리 생성

**사용자 설치 흐름:**
```bash
brew install biddan606/tap/asksh
asksh config          # 설정 마법사 실행
asksh "현재 브랜치의 모든 커밋 로그 보기"
```

---

## 결정 사항 기록

| # | 질문 | 결정 |
|---|---|---|
| 1 | LLM 프롬프트 언어 | 입력에서 자동 감지 (한국어/영어); 프롬프트와 UI는 동일 언어로 응답 |
| 2 | 쉘 컨텍스트 주입 | 예 — 모든 번역 프롬프트에 CWD, macOS 버전, 쉘 종류 주입 |
| 3 | 명령 히스토리 로그 | 선택적 활성화 (기본값 `enable = false`); 타임스탬프와 함께 `~/.config/asksh/history.log`에 저장. v1 마법사에서 제외, 수동으로 활성화 필요. |
| 4 | 별칭 제안 | v1 범위 외. 쉘 설정 변경의 복잡성을 피하기 위해 연기. |
