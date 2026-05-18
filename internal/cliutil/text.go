// Copyright 2026 nickvasilescu. Licensed under Apache-2.0. See LICENSE.

// Package cliutil contains shared helpers used across the CLI and MCP
// packages. Kept in its own package to avoid symbol collisions with
// agent-authored commands in package cli.
package cliutil

import (
	"regexp"
	"strings"
)

// LooksLikeAuthError checks if an error message body contains auth-related keywords.
func LooksLikeAuthError(msg string) bool {
	lower := strings.ToLower(msg)
	patterns := []string{
		`\bkey\b`,
		`\btoken\b`,
		`\bunauthorized\b`,
		`\bapi_key\b`,
		`missing.{0,20}key`,
		`required.{0,20}key`,
		`\bforbidden\b`,
		`\bauthenticat`,
		`\bcredential`,
	}
	for _, p := range patterns {
		if matched, _ := regexp.MatchString(p, lower); matched {
			return true
		}
	}
	return false
}

// SanitizeErrorBody truncates and strips credential-shaped strings from error output.
func SanitizeErrorBody(msg string) string {
	if len(msg) > 200 {
		msg = msg[:200] + "..."
	}
	credPatterns := regexp.MustCompile(`(?i)(sk-[a-zA-Z0-9]{8,}|sk_live_[a-zA-Z0-9]+|Bearer\s+[a-zA-Z0-9._\-]+|key=[a-zA-Z0-9._\-]+)`)
	msg = credPatterns.ReplaceAllString(msg, "[REDACTED]")
	return msg
}
