package model

import (
	"errors"
	"fmt"
	"os/exec"
)

type StorageLayout struct {
	Name         string         `json:"name"`
	RaidArrays   []*RaidArray   `json:"raid_arrays"`
	BlockDevices []*BlockDevice `json:"block_devices"`
	FileSystems  []*FileSystem  `json:"file_systems"`
}

type FileSystem struct {
	Name       string   `json:"name"`
	Format     string   `json:"format"`
	UUID       string   `json:"uuid"`
	MountPoint string   `json:"mount_point"`
	Options    []string `json:"format_options"`
}

var (
	ErrFailedPartitioning          = errors.New("failed partitioning")
	ErrBlockDeviceFailedValidation = errors.New("block device failed validation")
	ErrFailedExecution             = errors.New("failed execution")
	ErrArrayDeviceFailedValidation = errors.New("array device failed validation")
	ErrInvalidRaidType             = errors.New("invalid raid type")
	ErrInvalidDelimitedPartition   = errors.New("invalid delimited partition string")
)

func FailedExecutionError(cmdPath, errMsg string) error {
	return fmt.Errorf("FailedExecution %w : %s \"%s\"", ErrFailedExecution, cmdPath, errMsg)
}

func BlockDeviceFailedValidationError(bd *BlockDevice) error {
	return fmt.Errorf("BlockDeviceFailedValidation %w : %v", ErrBlockDeviceFailedValidation, bd)
}

func ArrayDeviceFailedValidationError(a *RaidArray) error {
	return fmt.Errorf("ArrayDeviceFailedValidation %w : %v", ErrArrayDeviceFailedValidation, a)
}

func FailedPartitioningError(position uint) error {
	return fmt.Errorf("FailedPartitioning %w : %d", ErrFailedPartitioning, position)
}

func InvalidRaidTypeError(raidType string) error {
	return fmt.Errorf("InvalidRaidType %w : %s", ErrInvalidRaidType, raidType)
}

func InvalidDelimitedPartitionError(delimitedString string) error {
	return fmt.Errorf("InvalidDelimitedPartition %w : %s", ErrInvalidDelimitedPartition, delimitedString)
}

func CallCommand(cmdName string, cmdOptions ...string) (out string, err error) {
	cmdPath, err := exec.LookPath(cmdName)
	if err != nil {
		return
	}

	cmd := exec.Command(cmdPath, cmdOptions...)

	outB, err := cmd.CombinedOutput()
	out = string(outB)

	if err != nil {
		err = FailedExecutionError(cmdPath, err.Error())
		return
	}

	return
}
