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
	Run: func(cmd *cobra.Command, _ []string) {
		ctx := command.NewContextWithLogger(cmd.Context(), logger)
		raidType := GetString(cmd, "raid-type")
		raidObjectType := GetString(cmd, "object-type")

		switch raidObjectType {
		case "vd":
			listVirtualDisks(ctx, raidType)
		case "pd":
			listPhysicalDisks(ctx, raidType)
		default:
			err := model.InvalidRaidObjectTypeError(raidObjectType)
			logger.Fatalw("invalid raid object type", "err", err, "raidObjectType", raidObjectType)
		}
	},
}

func init() {
	raidCmd.PersistentFlags().String("object-type", "vd", "Type of RAID objects to list: vd,pd")
	raidCmd.AddCommand(listRaidCmd)
}

func listVirtualDisks(ctx context.Context, raidType string) {
	virtualDisks, err := model.ListVirtualDisks(ctx, raidType)
	if err != nil {
		logger.Fatalw("failed to list virtual disks", "err", err, "raidType", raidType)
	}

	fmt.Println("id,name,raid-type")

	for _, vd := range virtualDisks {
		fmt.Printf("%s,%s,%s\n", vd.ID, vd.Name, vd.RaidType)
	}
}

func listPhysicalDisks(ctx context.Context, raidType string) {
	physicalDisks, err := model.ListPhysicalDisks(ctx, raidType)
	if err != nil {
		logger.Fatalw("failed to list physical disks", "err", err, "raidType", raidType)
	}

	fmt.Println("storage-controller-drive-id,drive-type,serial")

	for _, pd := range physicalDisks {
		fmt.Printf("%d,%s,%s\n", pd.StorageControllerDriveID, pd.Type, pd.Serial)
	}
}
