package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Stubbed inbox command to keep builds green when the full inbox implementation
// is not present in this checkout.
func newMailInboxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "inbox [session]",
		Short:  "Aggregate Agent Mail inbox (stubbed)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("mail inbox command is not available in this build")
		},
	}
	return cmd
}
