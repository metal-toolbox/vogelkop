package cmd

import (
	"context"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/metal-toolbox/vogelkop/internal/command"
	"github.com/metal-toolbox/vogelkop/pkg/model"
)

var deleteRaidCmd = &cobra.Command{
	Use:   "delete",
	Short: "Deletes a VirtualDisk (RAID array)",
	Long:  "Deletes a VirtualDisk (RAID array)",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := command.NewContextWithLogger(cmd.Context(), logger)
		raidType := GetString(cmd, "raid-type")
		deleteArray(ctx, raidType, GetString(cmd, "name"))
	},
}

func init() {
	deleteRaidCmd.PersistentFlags().String("name", "unknown", "Virtual Disk Name/ID")
	markFlagAsRequired(deleteRaidCmd, "name")

	raidCmd.AddCommand(deleteRaidCmd)
}

func deleteArray(ctx context.Context, raidType, arrayName string) {
	raidArray := model.RaidArray{
		Name: arrayName,
	}

	// If arrayName is actually an integer, populate that as the ControllerVirtualDiskID
	if id, err := strconv.Atoi(arrayName); err == nil {
		raidArray.ControllerVirtualDiskID = id
	}

	if out, err := raidArray.Delete(ctx, raidType); err != nil {
		logger.Fatalw("failed to create raid array", "err", err, "array", raidArray, "output", out)
	}
}
