package delete

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a resource",
	Long: `Delete a resource by ID.

Note: Delete is not yet supported by the SDK. This command will be
enabled once the platform API exposes delete endpoints.`,
	Args: cobra.NoArgs,
	RunE: run,
}

func run(_ *cobra.Command, _ []string) error {
	return fmt.Errorf("delete is not yet supported; the SDK does not expose delete endpoints")
}
