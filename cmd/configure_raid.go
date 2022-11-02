package cmd

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/metal-toolbox/vogelkop/internal/command"
	"github.com/metal-toolbox/vogelkop/pkg/model"
)

var configureRaidCmd = &cobra.Command{
	Use:   "configure-raid",
	Short: "Configures various types of RAID",
	Long:  "Configures various types of RAID",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := command.NewContextWithLogger(logger)

		if GetBool(cmd, "delete") {
			raidArray := model.RaidArray{
				Name: GetString(cmd, "name"),
			}

			if out, err := raidArray.Delete(ctx, GetString(cmd, "raid-type")); err != nil {
				logger.Fatalw("failed to create raid array", "err", err, "array", raidArray, "output", out)
			}
		} else {
			var blockDeviceIDs []int

			for _, d := range GetStringSlice(cmd, "devices") {
				intBlockDevice, err := strconv.Atoi(d)
				if err != nil {
					logger.Fatalw("failed to convert device id string to int", "err", err, "blockDeviceID", d)
				}

				blockDeviceIDs = append(blockDeviceIDs, intBlockDevice)
			}

			// TODO(splaspood) Handle looking up devices using ironlib/mvcli to generate this list?
			blockDevices, err := model.NewBlockDevicesFromPhysicalDeviceIDs(blockDeviceIDs...)
			if err != nil {
				logger.Fatalw("failed to gather block devices from physical ids", "err", err, "devices", blockDeviceIDs)
			}

			raidArray := model.RaidArray{
				Name:    GetString(cmd, "name"),
				Devices: blockDevices,
				Level:   GetString(cmd, "raid-level"),
			}

			if err = raidArray.Create(ctx, GetString(cmd, "raid-type")); err != nil {
				logger.Fatalw("failed to create raid array", "err", err, "array", raidArray)
			}
		}
	},
}

func init() {
	configureRaidCmd.PersistentFlags().String("raid-type", "linuxsw", "RAID Type (linuxsw,hardware)")
	configureRaidCmd.PersistentFlags().Bool("delete", false, "Delete virtual disk")

	configureRaidCmd.PersistentFlags().StringSlice("devices", []string{}, "List of underlying physical block devices.")
	markFlagAsRequired(configureRaidCmd, "devices")

	configureRaidCmd.PersistentFlags().String("raid-level", "1", "RAID Level")

	configureRaidCmd.PersistentFlags().String("name", "unknown", "RAID Volume Name")
	markFlagAsRequired(configureRaidCmd, "name")

	rootCmd.AddCommand(configureRaidCmd)
}
