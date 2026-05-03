# asksh

자연어(한국어/영어)를 쉘 명령어로 번역하고 안전하게 실행하는 macOS CLI 도구.

```
$ asksh "현재 브랜치의 마지막 5개 커밋 보기"
git log --oneline -n 5

[y] 실행  [n] 취소  [e] 수정  [?] 설명: y
```

현재 `asksh`는 **Release 파일이 아직 없어서 `curl ... releases/download/...tar.gz` 방식으로 설치하면 실패**합니다. 정식 릴리즈 전에는 직접 다운로드를 사용할 수 없고 소스 빌드를 사용하세요.

---

## 1. 필요한 것 설치

```bash
brew install git go ollama
```

이미 설치되어 있으면 확인만 하면 됩니다.

```bash
git --version
go version
ollama --version
```

`asksh`는 macOS용 CLI이고, 로컬 LLM으로 Ollama를 쓰거나 OpenAI API 키를 사용할 수 있습니다.

---

## 2. 저장소 다운로드

```bash
cd ~
git clone https://github.com/biddan606/asksh.git
cd asksh
```

---

## 3. 빌드

```bash
go build -o asksh .
```

빌드가 성공하면 현재 폴더에 `asksh` 실행 파일이 생깁니다.

확인:

```bash
ls -l asksh
./asksh --help
```

---

## 4. 실행 파일 설치

Apple Silicon Mac에서 Homebrew를 쓰고 있다면 보통 `/opt/homebrew/bin`이 PATH에 잡혀 있습니다.

```bash
cp asksh /opt/homebrew/bin/asksh
```

확인:

```bash
which asksh
asksh --help
```

만약 `/opt/homebrew/bin`이 없거나 권한 문제가 나면, 사용자 전용 경로에 설치하세요.

```bash
mkdir -p ~/.local/bin
cp asksh ~/.local/bin/asksh
chmod +x ~/.local/bin/asksh
```

그리고 `~/.zshrc`에 PATH 추가:

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

확인:

```bash
which asksh
asksh --help
```

---

## 5. Ollama 모델 준비

기본 사용 방식은 Ollama입니다.

```bash
ollama pull qwen2.5-coder:7b
```

Ollama 서버가 안 떠 있으면 실행:

```bash
ollama serve
```

이미 Ollama 앱이나 백그라운드 서비스가 실행 중이면 생략해도 됩니다.

---

## 6. asksh 초기 설정

```bash
asksh config
```

이 명령은 `~/.config/asksh/config.toml` 설정 파일을 만드는 대화형 마법사입니다.

```text
backend: ollama
host: http://localhost:11434
model: qwen2.5-coder:7b
```

---

## 7. 테스트 실행

먼저 안전하게 `--dry-run`으로 확인하세요.

```bash
asksh --dry-run "현재 폴더의 파일 목록 보여줘"
```

실제로 실행:

```bash
asksh "현재 폴더의 파일 목록 보여줘"
```

---

## 전체 명령어 한 번에

```bash
brew install git go ollama

cd ~
git clone https://github.com/biddan606/asksh.git
cd asksh

go build -o asksh .
cp asksh /opt/homebrew/bin/asksh

ollama pull qwen2.5-coder:7b

asksh config
asksh --dry-run "현재 폴더의 파일 목록 보여줘"
```

`/opt/homebrew/bin`에서 막히면 이 버전으로 설치하세요.

```bash
mkdir -p ~/.local/bin
cp asksh ~/.local/bin/asksh
chmod +x ~/.local/bin/asksh
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

---

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
