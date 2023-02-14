package cmd

import (
	"context"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/bmc-toolbox/common"
	"github.com/metal-toolbox/vogelkop/internal/command"
	"github.com/metal-toolbox/vogelkop/pkg/model"
)

var createRaidCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates a VirtualDisk from one or more PhysicalDisk(s)",
	Long:  "Creates a VirtualDisk from one or more PhysicalDisk(s)",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := command.NewContextWithLogger(cmd.Context(), logger)
		raidType := GetString(cmd, "raid-type")
		createArray(ctx, GetString(cmd, "name"), raidType, GetString(cmd, "raid-level"), GetStringSlice(cmd, "devices"))
	},
}

func init() {
	createRaidCmd.PersistentFlags().StringSlice("devices", []string{}, "List of underlying physical block devices.")
	markFlagAsRequired(createRaidCmd, "devices")
	createRaidCmd.PersistentFlags().String("raid-level", "1", "RAID Level")
	markFlagAsRequired(createRaidCmd, "raid-level")
	createRaidCmd.PersistentFlags().String("name", "unknown", "RAID Volume Name")
	markFlagAsRequired(createRaidCmd, "name")

	raidCmd.AddCommand(createRaidCmd)
}

func createArray(ctx context.Context, arrayName, raidType, raidLevel string, arrayDevices []string) {
	if raidType == "" {
		raidType = common.SlugRAIDImplLinuxSoftware
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
	case common.SlugRAIDImplLinuxSoftware:
		return processDevicesLinuxSw(arrayDevices)
	case common.SlugRAIDImplHardware:
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
