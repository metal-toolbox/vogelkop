package cmd

import (
	"github.com/metal-toolbox/vogelkop/pkg/model"
	"github.com/spf13/cobra"
)

var (
	partitionDiskCmd = &cobra.Command{
		Use:   "partition-disk",
		Short: "Partitions a block device",
		Long: "Partitions a block device with a GPT table",
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			s_partitions := GetStringSlice(cmd, "partitions")
			device := GetString(cmd, "device")

			for _, s_partition := range s_partitions {
				bd, err := model.NewBlockDevice(device)
				if err != nil {
					logger.Fatalw("Failed to create BlockDevice", "err", err, "device", device)
				}

				p, err := model.NewPartitionFromDelimited(s_partition, bd)
				if err != nil {
					logger.Fatalw("Failed to parse delimited partition data", "delimited_string", s_partition)
				}

				if out, err := p.Create(); err != nil {
					logger.Fatalw("failed to create partition", "err", err, "partition", p, "output", out)
				}
			}
		},
	}
)

func init() {
	partitionDiskCmd.PersistentFlags().String("device", "/dev/sda", "Device to be partitioned")
	markFlagAsRequired(partitionDiskCmd, "device")
	partitionDiskCmd.PersistentFlags().StringSlice("partitions", []string{}, "Partition Definitions Name:Position:Size:Type")
	rootCmd.AddCommand(partitionDiskCmd)
}