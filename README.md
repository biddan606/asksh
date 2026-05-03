# asksh

자연어(한국어/영어)를 쉘 명령어로 번역하고 안전하게 실행하는 macOS CLI 도구.

```
$ asksh "현재 브랜치의 마지막 5개 커밋 보기"
git log --oneline -n 5

[y] 실행  [n] 취소  [e] 수정  [?] 설명: y
```

## 요구 사항

- macOS (Apple Silicon 또는 Intel)
- Go (소스 빌드 시)
- [Ollama](https://ollama.ai) (로컬 LLM, 기본값) 또는 OpenAI API 키

## 설치

### Homebrew (권장)

> v0.1.0 정식 릴리즈 publish 전까지는 [소스 빌드](#소스-빌드)를 사용하세요.

```bash
brew tap biddan606/asksh
brew install asksh
```

### 직접 다운로드

> 정식 릴리즈 publish 전에는 사용할 수 없습니다. [소스 빌드](#소스-빌드)를 사용하세요.

[Releases](https://github.com/biddan606/asksh/releases) 페이지에서 최신 버전을 다운로드합니다.

```bash
# VERSION을 다운로드할 버전으로 교체하세요 (예: 0.1.0)
VERSION=0.1.0

# Apple Silicon
curl -L "https://github.com/biddan606/asksh/releases/download/v${VERSION}/asksh_${VERSION}_darwin_arm64.tar.gz" | tar xz
sudo mv asksh /usr/local/bin/

# Intel Mac
curl -L "https://github.com/biddan606/asksh/releases/download/v${VERSION}/asksh_${VERSION}_darwin_amd64.tar.gz" | tar xz
sudo mv asksh /usr/local/bin/
```

### 소스 빌드

```bash
git clone https://github.com/biddan606/asksh
cd asksh
go build -o asksh .
```

## 초기 설정

```bash
asksh config
```

대화형 마법사가 `~/.config/asksh/config.toml`을 생성합니다.

### Ollama 사용 시

```bash
# Ollama 설치 후 모델 다운로드
ollama pull qwen2.5-coder:7b

# Ollama 서버가 실행 중인지 확인 (앱이나 서비스로 이미 실행 중이면 생략)
ollama serve
```

### OpenAI 사용 시

마법사에서 API 키를 입력하거나 환경 변수로 설정합니다.

```bash
export OPENAI_API_KEY=sk-...
```

## 사용법

```
asksh <자연어 쿼리>       번역 및 실행
asksh config             설정 마법사
asksh --dry-run <쿼리>   쿼리/컨텍스트/안전성 정보만 출력 (LLM 번역·실행 없음)
asksh --ollama <쿼리>    Ollama 백엔드 강제
asksh --openai <쿼리>    OpenAI 백엔드 강제
asksh --version
asksh --help
```

## 예제

```bash
# 파일 조작
asksh "현재 폴더의 모든 .log 파일 삭제"
asksh "src 디렉토리를 src_backup으로 복사"

# Git
asksh "마지막 커밋 되돌리기"
asksh "현재 브랜치의 마지막 5개 커밋 보기"

# 프로세스
asksh "포트 8080을 사용 중인 프로세스 종료"
asksh "현재 실행 중인 Node 프로세스 목록"

# 영어도 지원
asksh "find all files larger than 100MB"
asksh "show disk usage by directory"

# 안전성 검사만 미리 확인 (번역·실행 없음)
asksh --dry-run "rm -rf ./node_modules"
```

## 안전 시스템

asksh는 2단계 안전 검사를 수행합니다.

**1단계 — 규칙 기반:** `rm -rf`, `sudo`, `curl | sh` 등 위험 패턴을 즉시 차단하거나 경고합니다.

**2단계 — LLM 검사:** 번역된 명령의 안전성을 LLM이 비동기로 평가합니다.

확인 프롬프트에서 `[e]`를 선택하면 명령을 직접 수정할 수 있고, `[?]`를 선택하면 LLM이 명령을 설명합니다.

### 종료 코드

| 코드 | 의미 |
|------|------|
| 0 | 성공 |
| 1 | 사용자 취소 |
| 2 | 안전 차단 |
| 3 | LLM 오류 |
| 4 | 설정 오류 |

## 설정 파일

`~/.config/asksh/config.toml`:

```toml
[backend]
default = "ollama"          # "ollama" | "openai"

[ollama]
host  = "http://localhost:11434"
model = "qwen2.5-coder:7b"

[openai]
api_key = ""                # 또는 OPENAI_API_KEY 환경 변수
model   = "gpt-4o-mini"

[safety]
require_confirmation = true
extra_llm_check      = true

[history]
enable = false              # true로 설정 시 ~/.config/asksh/history.log에 기록
```

## 문제 해결

**Ollama 연결 실패**

```
LLM error: Post "http://localhost:11434/api/chat": connection refused
```

Ollama가 실행 중인지 확인합니다.

```bash
ollama serve
```

**모델을 찾을 수 없음**

```bash
ollama pull qwen2.5-coder:7b
```

**OpenAI API 키 오류**

`asksh config`를 실행하거나 `OPENAI_API_KEY` 환경 변수를 설정합니다.

**색상이 표시되지 않음**

터미널이 색상을 지원하지 않는 환경에서는 `NO_COLOR=1 asksh ...`로 실행하거나, 환경 변수 `NO_COLOR`를 설정합니다.

## 라이선스

MIT
