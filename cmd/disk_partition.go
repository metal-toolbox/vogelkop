package cmd

import (
	"context"

	"github.com/metal-toolbox/vogelkop/pkg/model"
	"github.com/spf13/cobra"
)

var diskPartitionCommand = &cobra.Command{
	Use:   "partition",
	Short: "Partitions a disk with a GPT table",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, _ []string) {
		ctx := context.Background()

		partitions := GetStringSlice(cmd, "partitions")
		device := GetString(cmd, "device")

		for _, partition := range partitions {
			bd, err := model.NewBlockDevice(device)
			if err != nil {
				logger.Fatalw("Failed to create BlockDevice", "err", err, "device", device)
			}

			p, err := model.NewPartitionFromDelimited(partition, bd)
			if err != nil {
				logger.Fatalw("Failed to parse delimited partition data", "delimited_string", partition)
			}

			if out, err := p.Create(ctx); err != nil {
				logger.Fatalw("failed to create partition", "err", err, "partition", p, "output", out)
			}
		}
	},
}

func init() {
	diskPartitionCommand.PersistentFlags().String("device", "/dev/sda", "Device to be partitioned")
	markFlagAsRequired(diskPartitionCommand, "device")

	diskPartitionCommand.PersistentFlags().StringSlice("partitions", []string{}, "Partition Definitions Name:Position:Size:Type")

	diskCommand.AddCommand(diskPartitionCommand)

	deprecated := *diskPartitionCommand
	deprecated.Use = "partition-disk"
	deprecated.Deprecated = "use \"disk partition\""
	rootCmd.AddCommand(&deprecated)
}
