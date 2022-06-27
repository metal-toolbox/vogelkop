package cmd

import (
	"strconv"

	"github.com/spf13/cobra"
)

var (
	configureRaidCmd = &cobra.Command{
		Use:   "configure-raid",
		Short: "Configures software and hardware RAID",
		Long: "Configures software and hardware RAID",
		Run: func(cmd *cobra.Command, args []string) {
			configureRaid(
				GetStringSlice(cmd, "devices"),
				GetString(cmd, "name"),
				GetString(cmd, "raid-type"),
				GetString(cmd, "raid-level"),
			)
		},
	}
)

type RaidArray struct {
	Name string
	Devices []string
	Level string
}

func init() {
	configureRaidCmd.PersistentFlags().String("raid-type", "linuxsw", "RAID Type (linuxsw,dellperc,etc)")

	configureRaidCmd.PersistentFlags().StringSlice("devices", []string{}, "List of underlying physical volumes.")
	markFlagAsRequired("devices")

	configureRaidCmd.PersistentFlags().String("raid-level", "1", "RAID Level")

	configureRaidCmd.PersistentFlags().String("name", "unknown", "RAID Volume Name")
	markFlagAsRequired("name")
	
	rootCmd.AddCommand(configureRaidCmd)
}

func configureRaid(s_devices []string, name string, raid_type string, raid_level string) {
	// TODO(jwb) We should validate that the devices are valid / accessible here

	array := RaidArray{
		Name: name,
		Level: raid_level,
		Devices: s_devices,
	}

	switch raid_type {
	case "linuxsw":
		callLinuxSWRaid(array)
	}
}

func callLinuxSWRaid(array RaidArray) {
	cmd_args := []string{"--create", "/dev/md/" + array.Name,
		"--force", "--run", "--level", array.Level, "--raid-devices",
		strconv.Itoa(len(array.Devices))}
	cmd_args = append(cmd_args, array.Devices...)
	_, _ = callCommand("mdadm", cmd_args...)
}