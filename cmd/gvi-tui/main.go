// Command gvi-tui is the terminal user interface for Gastown Viewer Intent.
// It connects to the gvid daemon to display issue board and details.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/intent-solutions-io/gastown-viewer-intent/internal/tui"
)

// version is the binary's reported version. The dev default is "dev";
// release builds inject the tag via goreleaser ldflags
// (-X main.version={{.Version}}) — see .goreleaser.yaml.
var version = "dev"

func main() {
	apiURL := flag.String("api", "http://localhost:7070", "API server URL")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("gvi-tui version %s\n", version)
		os.Exit(0)
	}

	m := tui.New(*apiURL)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
