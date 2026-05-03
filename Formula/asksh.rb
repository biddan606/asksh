class Asksh < Formula
  desc "Translate natural language to shell commands using local Ollama or OpenAI"
  homepage "https://github.com/biddan606/asksh"
  version "0.1.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/biddan606/asksh/releases/download/v#{version}/asksh_#{version}_darwin_arm64.tar.gz"
      sha256 "REPLACE_WITH_ARM64_SHA256"
    else
      url "https://github.com/biddan606/asksh/releases/download/v#{version}/asksh_#{version}_darwin_amd64.tar.gz"
      sha256 "REPLACE_WITH_AMD64_SHA256"
    end
  end

  def install
    bin.install "asksh"
  end

  test do
    system "#{bin}/asksh", "--version"
  end
end
