# Formula implementation references:
# - https://docs.brew.sh/Formula-Cookbook
# - https://rubydoc.brew.sh/Formula
class Bump < Formula
  desc "Check and update package dependencies"
  homepage "https://github.com/MilosRandelovic/homebrew-bump"
  url "https://github.com/MilosRandelovic/homebrew-bump/archive/refs/tags/v2.2.1.tar.gz"
  sha256 "b540883c837feac0cc58cac1875689e48ada61e9fa107ee5cc55fe46482f5845"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(output: bin/"bump"), "."
    system "go", "build", "-mod=mod", "-o", bin/"bump-mcp", "github.com/MilosRandelovic/bump-core/v2/cmd/bump-mcp"
  end

  def caveats
    <<~EOS
      To register the MCP server with a supported client:
        claude mcp add bump -- #{opt_bin}/bump-mcp
        codex mcp add bump -- #{opt_bin}/bump-mcp
        grok mcp add bump -- #{opt_bin}/bump-mcp
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
