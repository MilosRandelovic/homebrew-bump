# Documentation:
# - https://docs.brew.sh/Formula-Cookbook
# - https://rubydoc.brew.sh/Formula
class Bump < Formula
  desc "A utility to check and update package dependencies"
  homepage "https://github.com/MilosRandelovic/homebrew-bump"
  url "https://github.com/MilosRandelovic/homebrew-bump/archive/v1.0.0.tar.gz"
  sha256 "c68fa309ee88982f8a6b64573e762860cbff3f396a7f0c50b1ebb43c20876840"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "-o", bin/"bump"
  end

  test do
    # Test version output
    assert_match "bump version", shell_output("#{bin}/bump -version")

    # Test help output
    assert_match "Usage: bump \\[options\\]", shell_output("#{bin}/bump -help")

    # Test error when no dependency files found
    assert_match "no package.json or pubspec.yaml found", shell_output("#{bin}/bump", 1)
  end
end
