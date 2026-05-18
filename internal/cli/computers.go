// Copyright 2026 nickvasilescu. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
)

func newComputersCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "computers",
		Short: "Provision and manage virtual computers",
	}

	cmd.AddCommand(newComputersCreateCmd(flags))
	cmd.AddCommand(newComputersDeleteCmd(flags))
	cmd.AddCommand(newComputersGetCmd(flags))
	cmd.AddCommand(newComputersAutoStopCmd(flags))
	cmd.AddCommand(newComputersStreamCmd(flags))
	cmd.AddCommand(newComputersBashExecuteCmd(flags))
	cmd.AddCommand(newComputersClickMouseCmd(flags))
	cmd.AddCommand(newComputersCloneComputerCmd(flags))
	cmd.AddCommand(newComputersDragMouseCmd(flags))
	cmd.AddCommand(newComputersExecExecutePythonCmd(flags))
	cmd.AddCommand(newComputersKeyPressCmd(flags))
	cmd.AddCommand(newComputersMoveComputerCmd(flags))
	cmd.AddCommand(newComputersResizeComputerCmd(flags))
	cmd.AddCommand(newComputersRestartComputerCmd(flags))
	cmd.AddCommand(newComputersScreenshotGetCmd(flags))
	cmd.AddCommand(newComputersScrollScrollCmd(flags))
	cmd.AddCommand(newComputersStartComputerCmd(flags))
	cmd.AddCommand(newComputersStopComputerCmd(flags))
	cmd.AddCommand(newComputersTypeTextCmd(flags))
	cmd.AddCommand(newComputersVncPasswordGetCmd(flags))
	cmd.AddCommand(newComputersWaitWaitCmd(flags))
	return cmd
}
