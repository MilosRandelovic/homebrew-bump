# Documentation:
# - https://docs.brew.sh/Formula-Cookbook
# - https://rubydoc.brew.sh/Formula
class Bump < Formula
  desc "A utility to check and update package dependencies"
  homepage "https://github.com/MilosRandelovic/homebrew-bump"
  url "https://github.com/MilosRandelovic/homebrew-bump/archive/v2.1.0.tar.gz"
  sha256 "ce82711c3b340e3afdaa0cefa677797cb8b7dc4bef1eb8158bcb62ed11145a3a"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(output: bin/"bump"), "."
    ENV["GOBIN"] = bin
    system "go", "install", "github.com/MilosRandelovic/bump-core/v2/cmd/bump-mcp@v2.2.0"
  end

  def caveats
    <<~EOS
      To register the MCP server with a supported client:
        claude mcp add bump -- #{opt_bin}/bump-mcp
        codex mcp add bump -- #{opt_bin}/bump-mcp
    EOS
  end

  test do
    # Test version output
    assert_match "bump version", shell_output("#{bin}/bump --version")

    # Test help output
    assert_match "Usage: bump [options]", shell_output("#{bin}/bump --help")

    # Test MCP server version output
    assert_match "bump-mcp version", shell_output("#{bin}/bump-mcp --version")

    # Test error when no dependency files found
    assert_match "no package.json or pubspec.yaml found", shell_output("#{bin}/bump 2>&1", 1)
  end
end
