package model

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/metal-toolbox/vogelkop/internal/command"
)

type Partition struct {
	Position          uint         `json:"position"`
	BlockDevice       *BlockDevice `json:"block_device"`
	Name              string       `json:"name"`
	Size              string       `json:"size"`
	Type              string       `json:"type"`
	FileSystem        string       `json:"file_system"`
	UUID              string       `json:"uuid"`
	MountPoint        string       `json:"mount_point"`
	FileSystemOptions []string     `json:"file_system_options"`
}

// NewPartitionFromDelimited returns a Partition based upon
// a delimited string value.
func NewPartitionFromDelimited(delimitedString string, bd *BlockDevice) (p *Partition, err error) {
	partition := strings.Split(delimitedString, ":")

	if len(partition) != 4 {
		err = InvalidDelimitedPartitionError(delimitedString)
		return
	}

	pos, err := strconv.Atoi(partition[1])
	if err != nil {
		return
	}

	p, err = NewPartition(partition[0], uint(pos), partition[2], partition[3])
	p.BlockDevice = bd

	return
}

func NewPartition(name string, position uint, size, ptype string) (p *Partition, err error) {
	if position < 1 || position > 128 {
		err = FailedPartitioningError(position)
		return
	}

	p = &Partition{
		Name:     name,
		Position: position,
		Size:     size,
		Type:     ptype,
	}

	return
}

// Format prepares a Partition on a given BlockDevice with a file system
// It returns an error object, or nil depending on the results.
func (p *Partition) Format() (out string, err error) {
	switch f := p.FileSystem; f {
	case "swap":
		out, err = command.Call("mkswap", p.BlockDevice.File)
	default:
		mkfsOptions := []string{"-F"}
		mkfsOptions = append(mkfsOptions, p.FileSystemOptions...)
		mkfsOptions = append(mkfsOptions, p.BlockDevice.File)
		out, err = command.Call("mkfs."+p.FileSystem, mkfsOptions...)
	}

	return
}

func (p *Partition) GetUUID() (string, error) {
	uuid, err := command.Call("blkid", "-s", "UUID", "-o", "value", p.BlockDevice.File)
	return strings.TrimRight(uuid, "\n"), err
}

func (p *Partition) Create() (out string, err error) {
	position := strconv.FormatInt(int64(p.Position), 10)
	out, err = command.Call("sgdisk",
		"-n", position+":0:"+p.Size,
		"-c", position+":"+p.Name,
		"-t", position+":"+p.Type,
		p.BlockDevice.File,
	)

	return
}

func (p *Partition) GetBlockDevice(device string) (systemDevice string) {
	position := strconv.FormatInt(int64(p.Position), 10)

	if strings.Contains(device, "loop") {
		systemDevice = p.GetLoopBlockDevice()
	} else {
		systemDevice = device + position
	}

	return
}

func (p *Partition) GetLoopBlockDevice() (systemDevice string) {
	position := strconv.FormatInt(int64(p.Position), 10)
	deviceFile := filepath.Base(p.BlockDevice.File)
	systemDevice = "/dev/mapper/" + deviceFile + "p" + position

	return
}
