class Bump < Formula
  desc "Check and update package dependencies"
  homepage "https://github.com/MilosRandelovic/homebrew-bump"
  url "https://github.com/MilosRandelovic/homebrew-bump/archive/refs/tags/v2.2.0.tar.gz"
  sha256 "c031a798bedec5a31cd44b6a31f42fd89c61d00377fffb049d3f395e4bc96fcf"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(output: bin/"bump"), "."
    system "go", "build", "-o", bin/"bump-mcp", "github.com/MilosRandelovic/bump-core/v2/cmd/bump-mcp"
  end

  def caveats
    <<~EOS
      To register the MCP server with a supported client:
        claude mcp add bump -- #{opt_bin}/bump-mcp
        codex mcp add bump -- #{opt_bin}/bump-mcp
    EOS
  end

  test do
    assert_match "bump version", shell_output("#{bin}/bump --version")
    assert_match "Usage: bump [options]", shell_output("#{bin}/bump --help")
    assert_match "bump-mcp version", shell_output("#{bin}/bump-mcp --version")
    assert_match "no package.json or pubspec.yaml found", shell_output("#{bin}/bump 2>&1", 1)
  end
end
