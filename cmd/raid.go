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

	rootCmd.AddCommand(raidCmd)
}

func getRaidType(cmd *cobra.Command) string {
	raidType := GetString(cmd, "raid-type")
	return raidType
}
