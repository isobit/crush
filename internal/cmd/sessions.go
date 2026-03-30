package cmd

import (
	"fmt"
	"time"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/db"
	"github.com/spf13/cobra"
)

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "List conversations",
	Long:  "List all conversations with their IDs, titles, and last updated time.",
	Args:  cobra.NoArgs,
	RunE:  runSessions,
}

func runSessions(cmd *cobra.Command, _ []string) error {
	dataDir, _ := cmd.Flags().GetString("data-dir")
	ctx := cmd.Context()

	if dataDir == "" {
		cfg, err := config.Init("", "", false)
		if err != nil {
			return fmt.Errorf("failed to initialize config: %w", err)
		}
		dataDir = cfg.Config().Options.DataDirectory
	}

	conn, err := db.Connect(ctx, dataDir)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close()

	sessions, err := db.New(conn).ListSessions(ctx)
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No sessions found.")
		return nil
	}

	w := cmd.OutOrStdout()
	for _, s := range sessions {
		title := s.Title
		if title == "" {
			title = "(untitled)"
		}
		updatedAt := time.Unix(s.UpdatedAt, 0).Format("2006-01-02 15:04")
		fmt.Fprintf(w, "%s  %-12s  %s\n", s.ID, updatedAt, title)
	}

	return nil
}
