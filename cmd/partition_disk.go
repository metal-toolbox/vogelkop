package cmd

import (
	"strings"
	"strconv"
	"os/exec"

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
			partitionDisk(cmd)
		},
	}
)

type Partition struct {
	Name string
	Position int
	Size string
	Type string
}

func init() {
	partitionDiskCmd.PersistentFlags().String("device", "/dev/sda", "Device to be partitioned")
	partitionDiskCmd.MarkPersistentFlagRequired("device")	
	partitionDiskCmd.PersistentFlags().StringSlice("partitions", []string{}, "Partition Definitions Name:Position:Size:Type")
	rootCmd.AddCommand(partitionDiskCmd)
}

func partitionDisk(cmd *cobra.Command) {
	s_parts, err := cmd.Flags().GetStringSlice("partitions")

	if err != nil {
		logger.Panicw("Error processing partitions parameter", "error", err)
	}

	partitions := processPartitions(s_parts)
	logger.Debugw("Processed partitions", "partitions", partitions)

	disk, _ := cmd.Flags().GetString("device")

	for _, partition := range partitions {
		callSgdisk(disk, partition)
	}
}

func callSgdisk(disk string, partition Partition) {
	position := strconv.FormatInt(int64(partition.Position),10)

	cmd := exec.Command("sgdisk",
		"-n", position + ":0:" + partition.Size,
		"-c", position + ":" + partition.Name,
		"-t", position + ":" + partition.Type,
		disk,
	)

	logger.Debugw("running sgdisk", "cmd", cmd)

	if out, err := cmd.CombinedOutput(); err != nil {
		logger.Debugf("%s\n", out)
		logger.Fatalw("Failed to run sgdisk",
			"partition", partition, "cmd", cmd, "err", err, "stderr", out)
	}
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
			Position: p_pos,
			Size: s_partition[2],
			Type: s_partition[3],
		}

		partitions = append(partitions, partition)
	}

	return
}