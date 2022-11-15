package cmd

import (
	"context"
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
		ctx := command.NewContextWithLogger(cmd.Context(), logger)
		raidType := GetString(cmd, "raid-type")
		if raidType == "" {
			raidType = "linuxsw"
		}

		if GetBool(cmd, "delete") {
			deleteArray(ctx, raidType, GetString(cmd, "name"))
		} else {
			createArray(ctx, GetString(cmd, "name"), raidType, GetString(cmd, "raid-level"), GetStringSlice(cmd, "devices"))
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

func deleteArray(ctx context.Context, raidType, arrayName string) {
	raidArray := model.RaidArray{
		Name: arrayName,
	}

	if out, err := raidArray.Delete(ctx, raidType); err != nil {
		logger.Fatalw("failed to create raid array", "err", err, "array", raidArray, "output", out)
	}
}

func createArray(ctx context.Context, arrayName, raidType, raidLevel string, arrayDevices []string) {
	if raidType == "" {
		raidType = "linuxsw"
	}

	raidArray := model.RaidArray{
		Name:  arrayName,
		Level: raidLevel,
	}

	raidArray.Devices = processDevices(arrayDevices, raidType)

	if err := raidArray.Create(ctx, raidType); err != nil {
		logger.Fatalw("failed to create raid array", "err", err, "array", raidArray)
	}
}

func processDevices(arrayDevices []string, raidType string) []*model.BlockDevice {
	switch raidType {
	case "linuxsw":
		return processDevicesLinuxSw(arrayDevices)
	case "hardware":
		return processDevicesHardware(arrayDevices)
	default:
		err := model.InvalidRaidTypeError(raidType)
		if err != nil {
			logger.Fatalw("invalid raid type", "err", err, "raidType", raidType)
		}

		return nil
	}
}

func processDevicesLinuxSw(arrayDevices []string) []*model.BlockDevice {
	blockDevices, err := model.NewBlockDevices(arrayDevices...)
	if err != nil {
		logger.Fatalw("Failed to GetBlockDevices", "err", err, "devices", arrayDevices)
	}

	return blockDevices
}

func processDevicesHardware(arrayDevices []string) []*model.BlockDevice {
	var blockDeviceIDs []int

	for _, d := range arrayDevices {
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

	return blockDevices
}
