package cmd

import (
	"github.com/spf13/cobra"
)

var partitionCommand = &cobra.Command{
	Use:   "partition",
	Short: "Modifies partitions",
}

func init() {
	rootCmd.AddCommand(partitionCommand)
}
