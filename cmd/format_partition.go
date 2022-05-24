package cmd

import (
	"github.com/spf13/cobra"
)

var (
	formatPartitionCmd = &cobra.Command{
		Use:   "format-partition",
		Short: "Formats a partition",
		Long: "Formats a partition with your choice of filesystem",
		Run: func(cmd *cobra.Command, args []string) {
			formatPartition()
		},
	}
)

func init() {
	formatPartitionCmd.PersistentFlags().String("disk", "/dev/sda", "Disk containing partition")
	formatPartitionCmd.PersistentFlags().Int("partition", 1, "Partition number to be formatted")
	formatPartitionCmd.PersistentFlags().String("filesystem", "ext4", "Filesystem to be applied to the partition")
	rootCmd.AddCommand(formatPartitionCmd)
}

func formatPartition() {
	logger.Infow("Configuring Disk /w Partitions")
}