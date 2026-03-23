class {{ .Name | title }} < Formula
  desc "{{ .Description }}"
  homepage "{{ .Homepage }}"
  version "{{ .Version }}"

  # Support for multi-architecture builds
  on_macos do
    if Hardware::CPU.intel?
      url "{{ .Assets.DarwinAMD64.URL }}"
      sha256 "{{ .Assets.DarwinAMD64.SHA256 }}"
    elsif Hardware::CPU.arm?
      url "{{ .Assets.DarwinARM64.URL }}"
      sha256 "{{ .Assets.DarwinARM64.SHA256 }}"
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "{{ .Assets.LinuxAMD64.URL }}"
      sha256 "{{ .Assets.LinuxAMD64.SHA256 }}"
    elsif Hardware::CPU.arm?
      url "{{ .Assets.LinuxARM64.URL }}"
      sha256 "{{ .Assets.LinuxARM64.SHA256 }}"
    end
  end

  def install
    # Installation instructions for the binary
    bin.install "{{ .BinaryName }}"
  end

  test do
    system "#{bin}/{{ .BinaryName }}", "--version"
  end
end
