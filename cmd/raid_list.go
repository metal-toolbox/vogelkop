package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/metal-toolbox/vogelkop/internal/command"
	"github.com/metal-toolbox/vogelkop/pkg/model"
)

var listRaidCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists existing VirtualDisk(s) (RAID arrays)",
	Long:  "Lists existing VirtualDisk(s) (RAID arrays)",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := command.NewContextWithLogger(cmd.Context(), logger)
		raidType := getRaidType(cmd)
		listVirtualDisks(ctx, raidType)
	},
}

func init() {
	raidCmd.AddCommand(listRaidCmd)
}

func listVirtualDisks(ctx context.Context, raidType string) {
	virtualDisks, err := model.ListVirtualDisks(ctx, raidType)
	if err != nil {
		logger.Fatalw("failed to list virtual disks", "err", err, "raidType", raidType)
	}

	for _, vd := range virtualDisks {
		fmt.Printf("%s,%s,%s\n", vd.ID, vd.Name, vd.RaidType)
	}
}
