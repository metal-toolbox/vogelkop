package cmd

import (
	"github.com/spf13/cobra"

	"github.com/metal-toolbox/vogelkop/pkg/model"
)

var partitionDiskCmd = &cobra.Command{
	Use:   "partition-disk",
	Short: "Partitions a block device",
	Long:  "Partitions a block device with a GPT table",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
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

			if out, err := p.Create(); err != nil {
				logger.Fatalw("failed to create partition", "err", err, "partition", p, "output", out)
			}
		}
	},
}

func init() {
	partitionDiskCmd.PersistentFlags().String("device", "/dev/sda", "Device to be partitioned")
	markFlagAsRequired(partitionDiskCmd, "device")
	partitionDiskCmd.PersistentFlags().StringSlice("partitions", []string{}, "Partition Definitions Name:Position:Size:Type")
	rootCmd.AddCommand(partitionDiskCmd)
}
