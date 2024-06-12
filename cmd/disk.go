package cmd

import (
	"github.com/spf13/cobra"
)

var diskCommand = &cobra.Command{
	Use:   "disk",
	Short: "Modifies disks",
}

func init() {
	rootCmd.AddCommand(diskCommand)
}
