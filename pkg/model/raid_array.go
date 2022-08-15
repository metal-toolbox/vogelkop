package model

import (
	"context"
	"strconv"

	"github.com/metal-toolbox/vogelkop/internal/command"
)

type RaidArray struct {
	Name    string         `json:"name"`
	Level   string         `json:"level"`
	Devices []*BlockDevice `json:"devices"`
}

// GetDeviceFiles returns a slice of strings with all the device files
// that make up the given RaidArray.
// It returns a slice of strings and possibly an error.
func (a *RaidArray) GetDeviceFiles() (deviceFiles []string, err error) {
	for _, dev := range a.Devices {
		deviceFiles = append(deviceFiles, dev.File)
	}

	return
}

// ValidateDevices validates that each block device is 'valid' by calling
// Validate on each BlockDevice.
// It returns false if any of the underlying calls to Validate() are false.
func (a *RaidArray) ValidateDevices() (valid bool) {
	for _, bd := range a.Devices {
		if !bd.Validate() {
			return false
		}
	}

	return true
}

func (a *RaidArray) Create(ctx context.Context, raidType string) (err error) {
	if !a.ValidateDevices() {
		err = ArrayDeviceFailedValidationError(a)
		return
	}

	switch raidType {
	case "linuxsw":
		err = a.CreateLinux(ctx)
	default:
		err = InvalidRaidTypeError(raidType)
	}

	return
}

func (a *RaidArray) DeleteLinux(ctx context.Context) (out string, err error) {
	out, err = command.Call(ctx, "mdadm", "--manage", "--stop", "/dev/md/"+a.Name)
	return
}

func (a *RaidArray) Delete(ctx context.Context, raidType string) (out string, err error) {
	switch raidType {
	case "linuxsw":
		out, err = a.DeleteLinux(ctx)
	default:
		err = InvalidRaidTypeError(raidType)
	}

	return
}

func (a *RaidArray) CreateLinux(ctx context.Context) (err error) {
	deviceFiles, err := a.GetDeviceFiles()
	if err != nil {
		return
	}

	cmdArgs := []string{
		"--create", "/dev/md/" + a.Name,
		"--force", "--run", "--level", a.Level, "--raid-devices",
		strconv.Itoa(len(a.Devices)),
	}
	cmdArgs = append(cmdArgs, deviceFiles...)
	_, err = command.Call(ctx, "mdadm", cmdArgs...)

	return
}
