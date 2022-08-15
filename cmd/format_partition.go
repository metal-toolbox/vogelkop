package cmd

import (
	"github.com/metal-toolbox/vogelkop/pkg/model"
	"github.com/spf13/cobra"
)

var formatPartitionCmd = &cobra.Command{
	Use:   "format-partition",
	Short: "Formats a partition",
	Long:  "Formats a partition with your choice of filesystem",
	Run: func(cmd *cobra.Command, args []string) {
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
				File: getPartitionBlockDevice(GetString(cmd, "device"), partition),
			}
		}

		if _, err := partition.Format(); err != nil {
			logger.Fatalw("failed to format partition", "err", err, "partition", partition)
		}
	},
}

func init() {
	formatPartitionCmd.PersistentFlags().String("device", "", "Block device")
	formatPartitionCmd.PersistentFlags().String("filesystem-device", "", "Filesystem Block device")

	formatPartitionCmd.PersistentFlags().Uint("partition", 0, "Partition number")

	formatPartitionCmd.PersistentFlags().String("format", "ext4", "Filesystem to be applied to the partition")
	markFlagAsRequired(formatPartitionCmd, "format")

	formatPartitionCmd.PersistentFlags().String("mount-point", "/", "Filesystem mount point")
	formatPartitionCmd.PersistentFlags().StringSlice("options", []string{}, "Filesystem creation options")
	rootCmd.AddCommand(formatPartitionCmd)
}
