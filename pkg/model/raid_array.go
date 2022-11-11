package model

import (
	"context"
	"strconv"

	"github.com/bmc-toolbox/common"
	"github.com/metal-toolbox/ironlib"
	"github.com/metal-toolbox/ironlib/actions"
	"github.com/metal-toolbox/ironlib/model"
	"github.com/metal-toolbox/vogelkop/internal/command"
)

type RaidArray struct {
	Name                    string         `json:"name"`
	Level                   string         `json:"level"`
	Devices                 []*BlockDevice `json:"devices"`
	ControllerVirtualDiskID int            `json:"controller_virtual_disk_id"`
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
		return a.CreateLinux(ctx)
	case "hardware":
		return a.CreateHardware(ctx)
	default:
		err = InvalidRaidTypeError(raidType)
		return
	}
}

func (a *RaidArray) DeleteLinux(ctx context.Context) (out string, err error) {
	out, err = command.Call(ctx, "mdadm", "--manage", "--stop", "/dev/md/"+a.Name)
	return
}

func (a *RaidArray) Delete(ctx context.Context, raidType string) (out string, err error) {
	switch raidType {
	case "linuxsw":
		return a.DeleteLinux(ctx)
	case "hardware":
		return a.DeleteHardware(ctx)
	default:
		err = InvalidRaidTypeError(raidType)
		return
	}
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

func (a *RaidArray) CreateHardware(ctx context.Context) (err error) {
	logrusLogger, err := command.ZapToLogrus(ctx)
	if err != nil {
		return
	}

	device, err := ironlib.New(logrusLogger)
	if err != nil {
		return
	}

	hardware, err := device.GetInventory(ctx, true)
	if err != nil {
		return
	}

	options := &model.CreateVirtualDiskOptions{
		RaidMode:        a.Level,
		PhysicalDiskIDs: []uint{0, 1},
		Name:            a.Name,
		BlockSize:       64,
	}

	// TODO(splaspood) We should pass the storage controller down here vs assuming
	for _, sc := range hardware.StorageControllers {
		if sc.Vendor == common.VendorMarvell {
			err = actions.CreateVirtualDisk(ctx, sc, options)
		}
	}

	return err
}

func (a *RaidArray) DeleteHardware(ctx context.Context) error {
	logrusLogger, err := command.ZapToLogrus(ctx)
	if err != nil {
		return err
	}

	device, err := ironlib.New(logrusLogger)
	if err != nil {
		return err
	}

	hardware, err := device.GetInventory(ctx, true)
	if err != nil {
		return err
	}

	for _, sc := range hardware.StorageControllers {
		if sc.Vendor == common.VendorMarvell {
			var vds []*common.VirtualDisk
			vds, err = actions.ListVirtualDisks(ctx, sc)
			if err != nil {
				return err
			}

			for _, vd := range vds {
				if vd.Name == a.Name {
					options := &model.DestroyVirtualDiskOptions{
						VirtualDiskID: a.ControllerVirtualDiskID,
					}

					err = actions.DestroyVirtualDisk(ctx, sc, options)
					return err
				}
			}
		}
	}

	return VirtualDiskNotFoundError(a)
}
