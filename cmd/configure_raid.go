package cmd

import (
	"os/exec"
	"strconv"

	"github.com/spf13/cobra"
)

var (
	configureRaidCmd = &cobra.Command{
		Use:   "configure-raid",
		Short: "Configures software and hardware RAID",
		Long: "Configures software and hardware RAID",
		Run: func(cmd *cobra.Command, args []string) {
			configureRaid(cmd)
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
	configureRaidCmd.MarkPersistentFlagRequired("devices")

	configureRaidCmd.PersistentFlags().String("raid-level", "1", "RAID Level")

	configureRaidCmd.PersistentFlags().String("name", "unknown", "RAID Volume Name")
	configureRaidCmd.MarkPersistentFlagRequired("name")

	rootCmd.AddCommand(configureRaidCmd)
}

func configureRaid(cmd *cobra.Command) {
	s_devices, err := cmd.Flags().GetStringSlice("devices")

	if err != nil {
		logger.Panicw("Error processing devices parameter", "error", err)
	}

	// TODO(jwb) We should validate that the devices are valid / accessible here

	array := RaidArray{
		Name: GetString(cmd, "name"),
		Level: GetString(cmd, "raid-level"),
		Devices: s_devices,
	}

	raidType := GetString(cmd, "raid-type")

	switch raidType {
	case "linuxsw":
		callLinuxSWRaid(array)
	}
}

func callLinuxSWRaid(array RaidArray) {
	cmd_args := []string{"--create", "/dev/md/" + array.Name,
		"--force", "--run", "--level", array.Level, "--raid-devices",
		strconv.Itoa(len(array.Devices))}
	cmd_args = append(cmd_args, array.Devices...)
	cmd := exec.Command("mdadm", cmd_args...)
	
	logger.Debugw("running mdadm", "cmd", cmd)

	if out, err := cmd.CombinedOutput(); err != nil {
		logger.Debugf("%s\n", out)
		logger.Fatalw("Failed to run mdadm",
			"array", array, "cmd", cmd, "err", err, "stderr", out)
	}
}