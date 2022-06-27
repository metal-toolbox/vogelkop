package cmd

import (
	"strings"
	"strconv"

	"github.com/spf13/cobra"
	// diskfs "github.com/diskfs/go-diskfs"
)

var (
	partitionDiskCmd = &cobra.Command{
		Use:   "partition-disk",
		Short: "Partitions a block device",
		Long: "Partitions a block device with a GPT table",
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			partitionDisk(GetString(cmd, "device"), GetStringSlice(cmd, "partitions"))
		},
	}
)

type Partition struct {
	Name string
	Position uint
	Size string
	Type string
	Format string
	FormatOptions []string
	BlockDevice string
	UUID string
	MountPoint string
}

func init() {
	partitionDiskCmd.PersistentFlags().String("device", "/dev/sda", "Device to be partitioned")
	markFlagAsRequired("device")
	partitionDiskCmd.PersistentFlags().StringSlice("partitions", []string{}, "Partition Definitions Name:Position:Size:Type")
	rootCmd.AddCommand(partitionDiskCmd)
}

func partitionDisk(device string, s_parts []string) {
	partitions := processPartitions(s_parts)
	logger.Debugw("Processed partitions", "partitions", partitions)

	for _, partition := range partitions {
		callSgdisk(device, partition)
	}
}

func callSgdisk(disk string, partition Partition) {
	position := strconv.FormatInt(int64(partition.Position),10)

	_, _ = callCommand("sgdisk",
		"-n", position + ":0:" + partition.Size,
		"-c", position + ":" + partition.Name,
		"-t", position + ":" + partition.Type,
		disk,
	)
}

func processPartitions(s_partitions []string) (partitions []Partition) {
	for _, raw_partition := range s_partitions {
		s_partition := strings.Split(raw_partition, ":")
		p_pos, _ := strconv.Atoi(s_partition[1])

		if p_pos < 1 || p_pos > 128 {
			logger.Fatalw("A partition position must be >= 1 && <= 128", "pos", p_pos)
		}

		partition := Partition{
			Name: s_partition[0],
			Position: uint(p_pos),
			Size: s_partition[2],
			Type: s_partition[3],
		}

		partitions = append(partitions, partition)
	}

	return
}