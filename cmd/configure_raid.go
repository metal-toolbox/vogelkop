package cmd

import (
	"github.com/metal-toolbox/vogelkop/pkg/model"
	"github.com/spf13/cobra"
)

var (
	configureRaidCmd = &cobra.Command{
		Use:   "configure-raid",
		Short: "Configures various types of RAID",
		Long:  "Configures various types of RAID",
		Run: func(cmd *cobra.Command, args []string) {
			blockDeviceFiles := GetStringSlice(cmd, "devices")

			blockDevices, err := model.NewBlockDevices(blockDeviceFiles...)
			if err != nil {
				logger.Fatalw("Failed to GetBlockDevices", "err", err, "devices", blockDeviceFiles)
			}

			raidArray := model.RaidArray{
				Name:    GetString(cmd, "name"),
				Devices: blockDevices,
				Level:   GetString(cmd, "raid-level"),
			}

			if err = raidArray.Create(GetString(cmd, "raid-type")); err != nil {
				logger.Fatalw("failed to create raid array", "err", err, "array", raidArray)
			}
		},
	}
)

func init() {
	configureRaidCmd.PersistentFlags().String("raid-type", "linuxsw", "RAID Type (linuxsw,dellperc,etc)")

	configureRaidCmd.PersistentFlags().StringSlice("devices", []string{}, "List of underlying physical volumes.")
	markFlagAsRequired(configureRaidCmd, "devices")

	configureRaidCmd.PersistentFlags().String("raid-level", "1", "RAID Level")

	configureRaidCmd.PersistentFlags().String("name", "unknown", "RAID Volume Name")
	markFlagAsRequired(configureRaidCmd, "name")

	rootCmd.AddCommand(configureRaidCmd)
}
