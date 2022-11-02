package model

import (
	"os"
	"path/filepath"
)

type BlockDevice struct {
	WWN                        string       `json:"wwn"`
	File                       string       `json:"file"`
	ControllerPhysicalDeviceID int          `json:"controller_physical_device_id"`
	Partitions                 []*Partition `json:"partitions"`
}

func NewBlockDevice(file string) (bd *BlockDevice, err error) {
	bd = &BlockDevice{
		ControllerPhysicalDeviceID: -1,
		File:                       file,
	}

	if !bd.Validate() {
		err = BlockDeviceFailedValidationError(bd)
		return
	}

	return
}

func NewBlockDeviceFromPhysicalDeviceID(physicalDeviceID int) (bd *BlockDevice, err error) {
	bd = &BlockDevice{
		ControllerPhysicalDeviceID: physicalDeviceID,
	}

	if !bd.Validate() {
		err = BlockDeviceFailedValidationError(bd)
		return
	}

	return
}

// NewBlockDevices returns a slice of BlockDevice(s) for the supplied
// slice of strings listing device files.
func NewBlockDevices(devices ...string) (blockDevices []*BlockDevice, err error) {
	for _, dev := range devices {
		bd, bdErr := NewBlockDevice(dev)
		if err != nil {
			return blockDevices, bdErr
		}

		blockDevices = append(blockDevices, bd)
	}

	return
}

// NewBlockDevicesFromPhysicalDeviceIDs returns a slice of BlockDevice(s) for the supplied
// slice of uints listing controller physical device ids.
func NewBlockDevicesFromPhysicalDeviceIDs(devices ...int) (blockDevices []*BlockDevice, err error) {
	for _, dev := range devices {
		bd, bdErr := NewBlockDeviceFromPhysicalDeviceID(dev)
		if err != nil {
			return blockDevices, bdErr
		}

		blockDevices = append(blockDevices, bd)
	}

	return
}

// Validate validates that the given block device is correct and accessible.
// It returns an bool indicating pass/fail.
func (b BlockDevice) Validate() bool {
	if b.ControllerPhysicalDeviceID >= 0 {
		// TODO(splaspood) validate this using ironlib
		return true
	} else if b.File != "" {
		resolvedPath, err := filepath.EvalSymlinks(b.File)
		if err != nil {
			return false
		}

		fi, err := os.Stat(resolvedPath) // Returns err if file is not accessible

		if os.IsNotExist(err) {
			return false
		}

		m := fi.Mode()

		return m&os.ModeDevice != 0
	}

	return false
}
