class Fog < Formula
  desc "Tool to manage your CloudFormation deployments"
  homepage "https://github.com/ArjenSchwarz/fog"

  # Use the latest release from GitHub
  version "0.1.0" # Update this with the actual version number when available

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/ArjenSchwarz/fog/releases/download/v#{version}/fog-darwin-arm64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256" # You'll need to calculate this
    else
      url "https://github.com/ArjenSchwarz/fog/releases/download/v#{version}/fog-darwin-amd64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256" # You'll need to calculate this
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/ArjenSchwarz/fog/releases/download/v#{version}/fog-linux-arm64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256" # You'll need to calculate this
    elsif Hardware::CPU.is_64_bit?
      url "https://github.com/ArjenSchwarz/fog/releases/download/v#{version}/fog-linux-amd64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256" # You'll need to calculate this
    else
      url "https://github.com/ArjenSchwarz/fog/releases/download/v#{version}/fog-linux-386.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256" # You'll need to calculate this
    end
  end

  def install
    # Extract the binary from the tarball and install it
    bin.install "fog"
  end

  test do
    system "#{bin}/fog", "--version"
  end
end
