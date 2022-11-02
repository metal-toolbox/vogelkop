package model

import (
	"errors"
	"fmt"
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
	ErrArrayDeviceFailedValidation = errors.New("array device failed validation")
	ErrInvalidRaidType             = errors.New("invalid raid type")
	ErrInvalidDelimitedPartition   = errors.New("invalid delimited partition string")
	ErrVirtualDiskNotFound         = errors.New("virtual disk not found")
)

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

func VirtualDiskNotFoundError(a *RaidArray) error {
	return fmt.Errorf("VirtualDiskNotFound %w : %v", ErrVirtualDiskNotFound, a)
}

// func (c contextKey) String() string {
// 	return "model context logger key " + string(c)
// }

// func NewContext(l *zap.SugaredLogger) context.Context {
// 	ctx := context.WithValue(context.Background(), contextLoggerKey, l)
// 	return ctx
// }

// func LoggerValueFromContext(ctx context.Context) *zap.SugaredLogger {
// 	logger, _ := ctx.Value(contextLoggerKey).(*zap.SugaredLogger)
// 	return logger
// }
