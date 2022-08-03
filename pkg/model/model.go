package model

import (
	// "bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	// diskfs "github.com/diskfs/go-diskfs"
	// losetup "github.com/freddierice/go-losetup/v2"
	// "path/filepath"
)

type StorageLayout struct {
	Name       string      `json:"name"`
	Volumes    []Volume    `json:"volumes"`
	RaidArrays []RaidArray `json:"raid_arrays"`
}

type BlockDevice struct {
	WWN  string `json:"wwn"`
	File string `json:"file"`
}

type RaidArray struct {
	Name    string        `json:"name"`
	Devices []*BlockDevice `json:"devices"`
	Level   string        `json:"level"`
}

type Volume struct {
	Device     BlockDevice `json:"device"`
	Partitions []Partition `json:"partitions"`
}

type Partition struct {
	Name          string   `json:"name"`
	Position      uint     `json:"position"`
	Size          string   `json:"size"`
	Type          string   `json:"type"`
	FileSystem        string   `json:"file_system"`
	FileSystemOptions []string `json:"file_system_options"`
	BlockDevice   *BlockDevice   `json:"block_device"`
	UUID          string   `json:"uuid"`
	MountPoint    string   `json:"mount_point"`
}

// NewPartitionFromDelimited returns a Partition based upon
// a delimited string value.
func NewPartitionFromDelimited(delimited_string string) (p *Partition, err error) {
	s_partition := strings.Split(delimited_string, ":")
	p_pos, err := strconv.Atoi(s_partition[1])

	if err != nil {
		return
	}

	p, err = NewPartition(s_partition[0], uint(p_pos), s_partition[2], s_partition[3])
	return
}

func NewPartition(name string, position uint, size string, ptype string) (p *Partition, err error) {
	if position < 1 || position > 128 {
		err = errors.New("Failed partitioning posititon: " + strconv.FormatUint(uint64(position), 10) + " A partition position must be >= 1 && <= 128")
		return
	}

	p = &Partition{
		Name: name,
		Position: position,
		Size: size,
		Type: ptype,
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
func NewBlockDevices(s_devices ...string) (block_devices []*BlockDevice, err error) {
	for _, dev := range s_devices {
		bd, bd_err := NewBlockDevice(dev)
		if err != nil {
			return block_devices, bd_err
		}

		block_devices = append(block_devices, bd)
	}

	return
}

// GetDeviceFiles returns a slice of strings with all the device files
// that make up the given RaidArray.
// It returns a slice of strings and possibly an error.
func (a RaidArray) GetDeviceFiles() (device_files []string, err error) {
	for _, dev := range a.Devices {
		device_files = append(device_files, dev.File)
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
	fi, err := os.Stat(b.File) // Returns err if file is not accessible

	if os.IsNotExist(err) {
		return false
	}

	m := fi.Mode()

	return m&os.ModeDevice != 0
}

// Format prepares a Partition on a given BlockDevice with a file system
// It returns an error object, or nil depending on the results.
func (p Partition) Format() (err error) {
	switch f := p.FileSystem; f {
	case "swap":
		_, err = callCommand("mkswap", p.BlockDevice.File)
	default:
		// TODO(jwb) Check for the existence of mkfs.FileSystem here
		mkfs_options := []string{"-F"}
		mkfs_options = append(mkfs_options, p.FileSystemOptions...)
		mkfs_options = append(mkfs_options, p.BlockDevice.File)
		_, err = callCommand("mkfs." + p.FileSystem, mkfs_options...)
	}

	return
}

func (p Partition) GetUUID() (string, error) {
	b_uuid, err := callCommand("blkid", "-s", "UUID", "-o", "value", p.BlockDevice.File)
	return strings.TrimRight(string(b_uuid), "\n"), err
}

func callCommand(cmd_name string, cmd_options ...string) (out string, err error) {
	cmd_path, err := exec.LookPath(cmd_name)
	if err != nil {
		return
	}
	cmd := exec.Command(cmd_path, cmd_options...)
	out_b, err := cmd.CombinedOutput()
	out = string(out_b)

	if err != nil {
		err = fmt.Errorf("failed to execute %s: %s", cmd_path, err.Error())
	}

	return
}

func (a RaidArray) Create(r_type string) (err error) {
	if !a.ValidateDevices() {
		err = errors.New("array devices failed validation")
		return
	}

	switch r_type {
	case "linuxsw":
		err = a.CreateLinux()
	}

	return
}

func (a RaidArray) CreateLinux() (err error) {
	device_files, err := a.GetDeviceFiles()

	if err != nil {
		return
	}

	cmd_args := []string{"--create", "/dev/md/" + a.Name,
		"--force", "--run", "--level", a.Level, "--raid-devices",
		strconv.Itoa(len(a.Devices))}
	cmd_args = append(cmd_args, device_files...)
	_, err = callCommand("mdadm", cmd_args...)

	return
}

func (p Partition) Create() (out string, err error) {
	position := strconv.FormatInt(int64(p.Position),10)
	out, err = callCommand("sgdisk",
		"-n", position + ":0:" + p.Size,
		"-c", position + ":" + p.Name,
		"-t", position + ":" + p.Type,
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

func (p Partition) GetLoopBlockDevice() (system_device string) {
	position := strconv.FormatInt(int64(p.Position),10)
	device_file := filepath.Base(p.BlockDevice.File)
	system_device = "/dev/mapper/" + device_file + "p" + position
	return
}
*/