package cmd

import (
	"github.com/bmc-toolbox/common"
	"github.com/spf13/cobra"
)

var raidCmd = &cobra.Command{
	Use:   "raid",
	Short: "Configures various types of RAID",
	Long:  "Configures various types of RAID",
}

func init() {
	raidCmd.PersistentFlags().String("raid-type", common.SlugRAIDImplLinuxSoftware, "RAID Type (linuxsw,hardware)")
	// markFlagAsRequired(raidCmd, "raid-type")

	rootCmd.AddCommand(raidCmd)
}

func getRaidType(cmd *cobra.Command) string {
	raidType := GetString(cmd, "raid-type")
	if raidType == "" {
		raidType = common.SlugRAIDImplLinuxSoftware
	}
	return raidType
}

func getRaidObjectType(cmd *cobra.Command) string {
	raidObjectType := GetString(cmd, "object-type")
	if raidObjectType == "" {
		raidObjectType = "vd"
	}
	return raidObjectType
}
