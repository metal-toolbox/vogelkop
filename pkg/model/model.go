package model

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"path/filepath"
)

type StorageLayout struct {
	Name         string         `json:"name"`
	RaidArrays   []*RaidArray   `json:"raid_arrays"`
	BlockDevices []*BlockDevice `json:"block_devices"`
	FileSystems  []*FileSystem  `json:"file_systems"`
}

type BlockDevice struct {
	WWN        string       `json:"wwn"`
	File       string       `json:"file"`
	Partitions []*Partition `json:"partitions"`
}

type RaidArray struct {
	Name    string         `json:"name"`
	Devices []*BlockDevice `json:"devices"`
	Level   string         `json:"level"`
}

type FileSystem struct {
	Name       string   `json:"name"`
	Format     string   `json:"format"`
	Options    []string `json:"format_options"`
	UUID       string   `json:"uuid"`
	MountPoint string   `json:"mount_point"`
}

type Partition struct {
	Name              string       `json:"name"`
	Position          uint         `json:"position"`
	Size              string       `json:"size"`
	Type              string       `json:"type"`
	FileSystem        string       `json:"file_system"`
	FileSystemOptions []string     `json:"file_system_options"`
	BlockDevice       *BlockDevice `json:"block_device"`
	UUID              string       `json:"uuid"`
	MountPoint        string       `json:"mount_point"`
}

// NewPartitionFromDelimited returns a Partition based upon
// a delimited string value.
func NewPartitionFromDelimited(delimitedString string, bd *BlockDevice) (p *Partition, err error) {
	partition := strings.Split(delimitedString, ":")
	pos, err := strconv.Atoi(partition[1])

	if err != nil {
		return
	}

	p, err = NewPartition(partition[0], uint(pos), partition[2], partition[3])
	p.BlockDevice = bd

	return
}

func NewPartition(name string, position uint, size string, ptype string) (p *Partition, err error) {
	if position < 1 || position > 128 {
		err = errors.New("Failed partitioning posititon: " + strconv.FormatUint(uint64(position), 10) + " A partition position must be >= 1 && <= 128")
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

func NewBlockDevice(file string) (bd *BlockDevice, err error) {
	bd = &BlockDevice{
		File: file,
	}

	if !bd.Validate() {
		err = errors.New("Block Device " + bd.File + " failed validation.")
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

// GetDeviceFiles returns a slice of strings with all the device files
// that make up the given RaidArray.
// It returns a slice of strings and possibly an error.
func (a RaidArray) GetDeviceFiles() (deviceFiles []string, err error) {
	for _, dev := range a.Devices {
		deviceFiles = append(deviceFiles, dev.File)
	}

	return
}

// ValidateDevices validates that each block device is 'valid' by calling
// Validate on each BlockDevice.
// It returns false if any of the underlying calls to Validate() are false.
func (a RaidArray) ValidateDevices() (valid bool) {
	for _, bd := range a.Devices {
		if !bd.Validate() {
			return false
		}
	}

	return true
}

// Validate validates that the given block device is correct and accessible.
// It returns an bool indicating pass/fail.
func (b BlockDevice) Validate() bool {
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

// Format prepares a Partition on a given BlockDevice with a file system
// It returns an error object, or nil depending on the results.
func (p Partition) Format() (out string, err error) {
	switch f := p.FileSystem; f {
	case "swap":
		out, err = CallCommand("mkswap", p.BlockDevice.File)
	default:
		// TODO(jwb) Check for the existence of mkfs.FileSystem here
		mkfsOptions := []string{"-F"}
		mkfsOptions = append(mkfsOptions, p.FileSystemOptions...)
		mkfsOptions = append(mkfsOptions, p.BlockDevice.File)
		out, err = CallCommand("mkfs."+p.FileSystem, mkfsOptions...)
	}

	return
}

func (p Partition) GetUUID() (string, error) {
	uuid, err := CallCommand("blkid", "-s", "UUID", "-o", "value", p.BlockDevice.File)
	return strings.TrimRight(string(uuid), "\n"), err
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
		err = fmt.Errorf("failed to execute %s: %s", cmdPath, err.Error())
	}

	return
}

func (a RaidArray) Create(raidType string) (err error) {
	if !a.ValidateDevices() {
		err = errors.New("array devices failed validation")
		return
	}

	switch raidType {
	case "linuxsw":
		err = a.CreateLinux()
	}

	return
}

func (a RaidArray) Disable(raidType string) (err error) {
	switch raidType {
	case "linuxsw":
		err = a.DisableLinux()
	}

	return
}

func (a RaidArray) DisableLinux() (err error) {
	_, err = CallCommand("mdadm", "--manage", "--stop", "/dev/md/" + a.Name)
	return
}

func (a RaidArray) Delete(raidType string) (err error) {
	switch raidType {
	case "linuxsw":
		err = a.DeleteLinux()
	}

	return
}

func (a RaidArray) DeleteLinux() (err error) {
	_, err = CallCommand("mdadm", "--manage", "--remove", "/dev/md/" + a.Name)
	return
}

func (a RaidArray) CreateLinux() (err error) {
	deviceFiles, err := a.GetDeviceFiles()

	if err != nil {
		return
	}

	cmdArgs := []string{"--create", "/dev/md/" + a.Name,
		"--force", "--run", "--level", a.Level, "--raid-devices",
		strconv.Itoa(len(a.Devices))}
	cmdArgs = append(cmdArgs, deviceFiles...)
	_, err = CallCommand("mdadm", cmdArgs...)

	return
}

func (p Partition) Create() (out string, err error) {
	position := strconv.FormatInt(int64(p.Position), 10)
	out, err = CallCommand("sgdisk",
		"-n", position+":0:"+p.Size,
		"-c", position+":"+p.Name,
		"-t", position+":"+p.Type,
		p.BlockDevice.File,
	)

	return
}

/*
func (p Partition) GetBlockDevice() (system_device string) {
 	position := strconv.FormatInt(int64(p.Position),10)

 	if strings.Contains(p.BlockDevice.File, "loop") {
 		system_device = p.GetLoopBlockDevice()
	} else {
		system_device = p.BlockDevice.File + position
	}

	return
}
*/

func (p Partition) GetLoopBlockDevice() (systemDevice string) {
	position := strconv.FormatInt(int64(p.Position),10)
	deviceFile := filepath.Base(p.BlockDevice.File)
	systemDevice = "/dev/mapper/" + deviceFile + "p" + position
	return
}
