package model

import (
	"context"
	"strconv"

	"github.com/bmc-toolbox/common"
	"github.com/metal-toolbox/ironlib"
	"github.com/metal-toolbox/ironlib/actions"
	"github.com/metal-toolbox/ironlib/model"
	"github.com/metal-toolbox/vogelkop/internal/command"
	"github.com/sirupsen/logrus"
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
	case common.SlugRAIDImplLinuxSoftware:
		return a.CreateLinux(ctx)
	case common.SlugRAIDImplHardware:
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
	case common.SlugRAIDImplLinuxSoftware:
		return a.DeleteLinux(ctx)
	case common.SlugRAIDImplHardware:
		return "", a.DeleteHardware(ctx)
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
	hardware, err := getIronlibInventory(ctx)

	options := &model.CreateVirtualDiskOptions{
		RaidMode:        a.Level,
		PhysicalDiskIDs: []uint{0, 1},
		Name:            a.Name,
		BlockSize:       64,
	}

	// TODO(splaspood) We should pass the storage controller down here vs assuming
	for _, sc := range hardware.StorageControllers {
		if sc.Vendor == common.VendorMarvell {
			var sca *actions.StorageControllerAction
			sca, err = getStorageControllerAction(ctx)
			if err != nil {
				return err
			}

			err = sca.CreateVirtualDisk(ctx, sc, options)
		}
	}

	return err
}

func (a *RaidArray) DeleteHardware(ctx context.Context) error {
	hardware, err := getIronlibInventory(ctx)
	if err != nil {
		return err
	}

	for _, sc := range hardware.StorageControllers {
		if sc.Vendor != common.VendorMarvell {
			continue
		}

		var vds []*common.VirtualDisk
		var sca *actions.StorageControllerAction
		sca, err = getStorageControllerAction(ctx)
		if err != nil {
			return err
		}

		vds, err = sca.ListVirtualDisks(ctx, sc)
		if err != nil {
			return err
		}

		for _, vd := range vds {
			if vd.Name != a.Name && vd.ID != strconv.Itoa(a.ControllerVirtualDiskID) {
				continue
			}

			options := &model.DestroyVirtualDiskOptions{
				VirtualDiskID: a.ControllerVirtualDiskID,
			}

			var sca *actions.StorageControllerAction
			sca, err = getStorageControllerAction(ctx)
			if err != nil {
				return err
			}

			err = sca.DestroyVirtualDisk(ctx, sc, options)
			return err
		}
	}

	return VirtualDiskNotFoundError(a)
}

func ListVirtualDisks(ctx context.Context, raidType string) (virtualDisks []*common.VirtualDisk, err error) {
	switch raidType {
	case common.SlugRAIDImplLinuxSoftware:
		return listVirtualDisksLinux(ctx)
	case common.SlugRAIDImplHardware:
		return listVirtualDisksHardware(ctx)
	default:
		err = InvalidRaidTypeError(raidType)
		return
	}
}

func ListPhysicalDisks(ctx context.Context, raidType string) (physicalDisks []*common.Drive, err error) {
	switch raidType {
	case common.SlugRAIDImplLinuxSoftware:
		return listPhysicalDisksLinux(ctx)
	case common.SlugRAIDImplHardware:
		return listPhysicalDisksHardware(ctx)
	default:
		err = InvalidRaidTypeError(raidType)
		return
	}
}

func listVirtualDisksLinux(_ context.Context) (virtualDisks []*common.VirtualDisk, err error) {
	// TODO(splaspood) Implement VD listing for mdadm

	// mdadm --misc --detail --export /dev/md/<name>*
	// Seems on my test ubuntu 20.04 host these end up named /dev/md/ROOT_0 (that _0)
	return
}

func listPhysicalDisksLinux(_ context.Context) (physicalDisks []*common.Drive, err error) {
	// TODO(splaspood) Implement PD listing for mdadm/noraid
	return
}

func listVirtualDisksHardware(ctx context.Context) (virtualDisks []*common.VirtualDisk, err error) {
	hardware, err := getIronlibInventory(ctx)
	if err != nil {
		return
	}

	for _, sc := range hardware.StorageControllers {
		if sc.Vendor != common.VendorMarvell {
			continue
		}

		var sca *actions.StorageControllerAction
		sca, err = getStorageControllerAction(ctx)
		if err != nil {
			return
		}

		virtualDisks, err = sca.ListVirtualDisks(ctx, sc)
		if err != nil {
			return
		}
	}

	return
}

func listPhysicalDisksHardware(ctx context.Context) (physicalDisks []*common.Drive, err error) {
	hardware, err := getIronlibInventory(ctx)
	if err != nil {
		return
	}

	for _, drive := range hardware.Drives {
		if drive.StorageControllerDriveID >= 0 {
			physicalDisks = append(physicalDisks, drive)
		}
	}

	return
}

func getIronlibInventory(ctx context.Context) (hardware *common.Device, err error) {
	logrusLogger, err := command.ZapToLogrus(ctx)
	if err != nil {
		return
	}

	device, err := ironlib.New(logrusLogger)
	if err != nil {
		return
	}

	hardware, err = device.GetInventory(ctx,
		actions.WithDynamicCollection(),
		actions.WithDisabledCollectorUtilities([]model.CollectorUtility{"hdparm"}),
	)

	return
}

func getStorageControllerAction(ctx context.Context) (sca *actions.StorageControllerAction, err error) {
	var logrusLogger *logrus.Logger
	logrusLogger, err = command.ZapToLogrus(ctx)
	if err != nil {
		return
	}

	sca = actions.NewStorageControllerAction(logrusLogger)
	return
}
