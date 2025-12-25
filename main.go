package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nicklasos/supervisord-tui/internal/ui"
)

var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "Show version information")
	configPath := flag.String("config", "", "Path to supervisord config file (default: auto-detect)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("god version %s\n", version)
		os.Exit(0)
	}

	var model *ui.Model
	var err error
	if *configPath != "" {
		model, err = ui.InitialModelWithConfig(*configPath)
	} else {
		model, err = ui.InitialModel()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing application: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
