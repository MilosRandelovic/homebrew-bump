# Documentation:
# - https://docs.brew.sh/Formula-Cookbook
# - https://rubydoc.brew.sh/Formula
class Bump < Formula
  desc "A utility to check and update package dependencies"
  homepage "https://github.com/MilosRandelovic/homebrew-bump"
  url "https://github.com/MilosRandelovic/homebrew-bump/archive/v1.3.0.tar.gz"
  sha256 "e2bf709fc70bf4a3698f5c00509aee2a7dc92dbe773b2fd3bf9dd760a5f1bf9c"
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
