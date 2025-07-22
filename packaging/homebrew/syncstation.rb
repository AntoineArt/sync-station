class Syncstation < Formula
  desc "CLI tool for syncing configuration files across computers using cloud storage"
  homepage "https://github.com/AntoineArt/syncstation"
  version "1.0.0"
  license "GPL-3.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/AntoineArt/syncstation/releases/download/v#{version}/syncstation-#{version}-macos-arm64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256_ARM64"
    else
      url "https://github.com/AntoineArt/syncstation/releases/download/v#{version}/syncstation-#{version}-macos-amd64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256_AMD64"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/example/syncstation/releases/download/v#{version}/syncstation-#{version}-linux-arm64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256_LINUX_ARM64"
    else
      url "https://github.com/AntoineArt/syncstation/releases/download/v#{version}/syncstation-#{version}-linux-amd64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256_LINUX_AMD64"
    end
  end

  def install
    if OS.mac?
      if Hardware::CPU.arm?
        bin.install "syncstation-macos-arm64" => "syncstation"
      else
        bin.install "syncstation-macos-amd64" => "syncstation"
      end
    else
      if Hardware::CPU.arm?
        bin.install "syncstation-linux-arm64" => "syncstation"
      else
        bin.install "syncstation-linux-amd64" => "syncstation"
      end
    end
  end

  test do
    system "#{bin}/syncstation", "--version"
    system "#{bin}/syncstation", "--help"
  end
end