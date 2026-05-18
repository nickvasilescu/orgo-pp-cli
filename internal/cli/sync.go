// Copyright 2026 nickvasilescu. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nickvasilescu/orgo-pp-cli/internal/store"
	"github.com/spf13/cobra"
)

func newSyncCmd(flags *rootFlags) *cobra.Command {
	var dbPath string

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync /workspaces to a local SQLite store for offline FTS",
		Long: `Fetch the user's workspaces (projects + nested desktops) from the Orgo API
and store each as a row in the local SQLite resources table. The store
powers the audit/grep action ledger; this command only refreshes the
workspace catalog so cross-workspace commands (fleet, idle, prune, cost)
can run without hitting the API for every lookup.`,
		Example: `  orgo-pp-cli sync
  orgo-pp-cli sync --db /tmp/orgo.db`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			c.NoCache = true

			if dbPath == "" {
				dbPath = defaultDBPath("orgo-pp-cli")
			}
			db, err := store.OpenWithContext(cmd.Context(), dbPath)
			if err != nil {
				return fmt.Errorf("opening local database: %w", err)
			}
			defer db.Close()

			started := time.Now()
			data, err := c.Get("/workspaces", nil)
			if err != nil {
				return fmt.Errorf("fetching /workspaces: %w", err)
			}

			items := extractWorkspaceItems(data)
			stored, _, err := db.UpsertBatch("workspaces", items)
			if err != nil {
				return fmt.Errorf("upserting workspaces: %w", err)
			}
			if err := db.SaveSyncState("workspaces", "", stored); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to save sync state: %v\n", err)
			}

			elapsed := time.Since(started)
			if flags.asJSON {
				return printJSONFiltered(cmd.OutOrStdout(), map[string]any{
					"resource":    "workspaces",
					"stored":      stored,
					"duration_ms": elapsed.Milliseconds(),
				}, flags)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "synced %d workspaces in %s\n", stored, elapsed.Round(time.Millisecond))
			return nil
		},
	}
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path (default: ~/.local/share/orgo-pp-cli/data.db)")
	return cmd
}

// extractWorkspaceItems pulls the projects array out of a /workspaces
// response. The Orgo API returns {"projects": [...]} so we look for that
// envelope first, then fall back to treating the whole body as an array.
func extractWorkspaceItems(data json.RawMessage) []json.RawMessage {
	var envelope struct {
		Projects []json.RawMessage `json:"projects"`
	}
	if err := json.Unmarshal(data, &envelope); err == nil && len(envelope.Projects) > 0 {
		return envelope.Projects
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err == nil {
		return arr
	}
	return nil
}

// parseSinceDuration converts human-friendly duration strings into a time.Time.
// Supported formats: "7d" (days), "24h" (hours), "30m" (minutes), "1w" (weeks).
func parseSinceDuration(s string) (time.Time, error) {
	re := regexp.MustCompile(`^(\d+)([dhwm])$`)
	matches := re.FindStringSubmatch(strings.TrimSpace(s))
	if matches == nil {
		return time.Time{}, fmt.Errorf("expected format like 7d, 24h, 1w, or 30m")
	}

	n, err := strconv.Atoi(matches[1])
	if err != nil {
		return time.Time{}, err
	}

	now := time.Now()
	switch matches[2] {
	case "d":
		return now.Add(-time.Duration(n) * 24 * time.Hour), nil
	case "h":
		return now.Add(-time.Duration(n) * time.Hour), nil
	case "w":
		return now.Add(-time.Duration(n) * 7 * 24 * time.Hour), nil
	case "m":
		return now.Add(-time.Duration(n) * time.Minute), nil
	default:
		return time.Time{}, fmt.Errorf("unknown unit %q", matches[2])
	}
}
