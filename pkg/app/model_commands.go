package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/jesseduffield/lazygit/pkg/ai"
	"github.com/jesseduffield/lazygit/pkg/config"
)

func runModelCommand(command, name string, out io.Writer) error {
	if command != "list" && (strings.TrimSpace(name) == "" || strings.HasPrefix(name, "-") || strings.ContainsAny(name, " \t\r\n")) {
		return fmt.Errorf("usage: lazygit-ai %s MODEL", command)
	}
	settings, err := config.LoadModelSettings()
	if err != nil {
		return err
	}
	client := ai.ModelClient{Endpoint: settings.AI.Endpoint}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	timeout := 15 * time.Second
	if command == "pull" {
		timeout = 30 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	switch command {
	case "pull":
		fmt.Fprintf(out, "Downloading %s from Ollama...\n", name)
		last := ""
		err := client.Pull(ctx, name, func(update ai.ModelProgress) {
			status := update.Status
			if update.Total > 0 {
				status = fmt.Sprintf("%s (%d%%)", status, update.Completed*100/update.Total)
			}
			if status != last {
				fmt.Fprintln(out, status)
				last = status
			}
		})
		if err != nil {
			return err
		}
		name = ai.ModelName(name)
	case "use", "list":
		models, err := client.List(ctx)
		if err != nil {
			return err
		}
		if command == "list" {
			fmt.Fprintf(out, "Configured model: %s (AI enabled: %t)\nConfig: %s\n", settings.AI.Model, settings.AI.Enabled && settings.AI.AutoGenerateCommitMessage, settings.Path)
			if len(models) == 0 {
				fmt.Fprintln(out, "No models installed. Run: lazygit-ai pull qwen2.5-coder:3b")
				return nil
			}
			table := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
			fmt.Fprintln(table, "MODEL\tSIZE (GB)\tSELECTED")
			for _, model := range models {
				selected := ""
				if ai.ModelName(model.Name) == ai.ModelName(settings.AI.Model) {
					selected = "*"
				}
				fmt.Fprintf(table, "%s\t%.2f\t%s\n", model.Name, float64(model.Size)/1e9, selected)
			}
			return table.Flush()
		}
		found := false
		for _, model := range models {
			if ai.ModelName(model.Name) == ai.ModelName(name) {
				name = model.Name
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("model %q is not installed; run: lazygit-ai pull %s", name, name)
		}
	default:
		return fmt.Errorf("unknown model command: %s", command)
	}
	if err := settings.SelectModel(name); err != nil {
		return err
	}
	fmt.Fprintf(out, "Active model: %s\nAI commit messages enabled. Config saved: %s\nOpen (or restart) lazygit-ai to use it.\n", name, settings.Path)
	return nil
}
