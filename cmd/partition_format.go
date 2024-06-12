package cmd

import (
	"context"

	"github.com/metal-toolbox/vogelkop/pkg/model"
	"github.com/spf13/cobra"
)

var partitionFormatPartitionCommand = &cobra.Command{
	Use:   "format",
	Short: "Formats a partition",
	Long:  "Formats a partition with your choice of filesystem",
	Run: func(cmd *cobra.Command, _ []string) {
		ctx := context.Background()

		if GetString(cmd, "device") == "" && GetString(cmd, "filesystem-device") == "" {
			logger.Fatal("Either --device or --filesystem-device are required.")
		}

		if GetString(cmd, "device") != "" && GetUint(cmd, "partition") == 0 {
			logger.Fatal("When using the --device parameter, the --partition number must be specified.")
		}

		pPosition := GetUint(cmd, "partition")
		filesystemDevice := GetString(cmd, "filesystem-device")

		partition := &model.Partition{
			Position:          pPosition,
			FileSystem:        GetString(cmd, "format"),
			FileSystemOptions: GetStringSlice(cmd, "options"),
			MountPoint:        GetString(cmd, "mount-point"),
		}

		if filesystemDevice != "" {
			partition.BlockDevice = &model.BlockDevice{
				File: filesystemDevice,
			}
		} else {
			partition.BlockDevice = &model.BlockDevice{
				File: partition.GetBlockDevice(GetString(cmd, "device")),
			}
		}

		if _, err := partition.Format(ctx); err != nil {
			logger.Fatalw("failed to format partition", "err", err, "partition", partition)
		}
	},
}

func init() {
	partitionFormatPartitionCommand.PersistentFlags().String("device", "", "Block device")
	partitionFormatPartitionCommand.PersistentFlags().String("filesystem-device", "", "Filesystem Block device")

	partitionFormatPartitionCommand.PersistentFlags().Uint("partition", 0, "Partition number")

	partitionFormatPartitionCommand.PersistentFlags().String("format", "ext4", "Filesystem to be applied to the partition")
	markFlagAsRequired(partitionFormatPartitionCommand, "format")

	partitionFormatPartitionCommand.PersistentFlags().String("mount-point", "/", "Filesystem mount point")
	partitionFormatPartitionCommand.PersistentFlags().StringSlice("options", []string{}, "Filesystem creation options")
	partitionCommand.AddCommand(partitionFormatPartitionCommand)

	rootCmd.AddCommand(&cobra.Command{
		Use:        "format-partition",
		Deprecated: "use \"partition format\"",
		Run:        partitionFormatPartitionCommand.Run,
	})
}
