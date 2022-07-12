package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

var (
	formatPartitionCmd = &cobra.Command{
		Use:   "format-partition",
		Short: "Formats a partition",
		Long: "Formats a partition with your choice of filesystem",
		Run: func(cmd *cobra.Command, args []string) {
			if (GetString(cmd, "device") == "" && GetString(cmd, "filesystem-device") == "") {
				logger.Fatal("Either --device or --filesystem-device are required.")
			}

			if (GetString(cmd, "device") != "" && GetUint(cmd, "partition") == 0) {
				logger.Fatal("When using the --device parameter, the --partition number must be specified.")
			}

			formatPartition(
				GetString(cmd, "device"),
				GetString(cmd, "filesystem-device"),
				GetUint(cmd, "partition"),
				GetString(cmd, "format"),
				GetString(cmd, "mount-point"),
				GetStringSlice(cmd, "options"),
			)
		},
	}
)

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

func formatPartition(device string, filesystem_device string, partition_number uint,
	format string, mount_point string, s_options []string) {

	partition := Partition{
		Position:      partition_number,
		Format:        format,
		FormatOptions: s_options,
		MountPoint:    mount_point,
	}

	if (filesystem_device != "") {
		partition.BlockDevice = filesystem_device
	} else {
		partition.BlockDevice = getPartitionBlockDevice(device, partition)
	}

	switch f := partition.Format; f {
	case "swap":
		_ = callCommand("mkswap", partition.BlockDevice)
	default:
		mkfs_options := []string{"-F"}
		mkfs_options = append(mkfs_options, partition.FormatOptions...)
		mkfs_options = append(mkfs_options, partition.BlockDevice)
		_ = callCommand("mkfs." + partition.Format, mkfs_options...)
	}

	partition.UUID = getBlockDeviceUUID(partition.BlockDevice)
}

func getBlockDeviceUUID(device string) (uuid string) {
	b_uuid := callCommand("blkid", "-s", "UUID", "-o", "value", device)
	return strings.TrimRight(string(b_uuid), "\n")
}