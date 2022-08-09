package model_test

import (
	"os"
	"path/filepath"
	"testing"
	"crypto/rand"
	"encoding/hex"

	diskfs "github.com/diskfs/go-diskfs"
	losetup "github.com/freddierice/go-losetup/v2"
	"github.com/metal-toolbox/vogelkop/pkg/model"
)

func cleanupTestDisk(image_name string, loopdev *losetup.Device) (err error) {
	if err = loopdev.Detach(); err != nil {
		return
	}

	if err = os.Remove(image_name); err != nil {
		return
	}

	return
}

func prepareTestDisk(size int64) (loopdev losetup.Device, disk_img string, err error) {
	disk_img = tempFileName("", "")

	var (
		disk_size int64 = size * 1024 * 1024
	)

	_, err = diskfs.Create(disk_img, disk_size, diskfs.Raw)
	if err != nil {
		return
	}

	loopdev, err = losetup.Attach(disk_img, 0, false)
	if err != nil {
		return
	}

	return
}

func createPartitions(bd *model.BlockDevice, partitions []*model.Partition) (out string, err error) {
	for _, p := range partitions {
		p.BlockDevice = bd

		out, err = p.Create()

		if err != nil {
			return
		}
	}

	// Since we are using loopback devices for this test, we need to use kpartx
	// to make the partition devices accessible
	out, err = kpartxAdd(bd.File)
	if err != nil {
		return
	}

	for _, p := range partitions {
		var partition_bd *model.BlockDevice

		partition_bd, err = model.NewBlockDevice(p.GetLoopBlockDevice())

		if err != nil {
			return
		}

		p.BlockDevice = partition_bd
	}

	return
}

func kpartxAdd(device string) (k_out string, err error) {
	k_out, err = model.CallCommand("kpartx", "-a", device)
	return
}

func kpartxDel(device string) (k_out string, err error) {
	k_out, err = model.CallCommand("kpartx", "-d", device)
	return
}

func tempFileName(prefix, suffix string) string {
	randBytes := make([]byte, 16)
	_, _ = rand.Read(randBytes)
	return filepath.Join(os.TempDir(), prefix+hex.EncodeToString(randBytes)+suffix)
}

func TestConfigureRaid(t *testing.T) {
	tests := []model.StorageLayout{
		{
			Name: "LinuxSoftwareRaid1",
			BlockDevices: []*model.BlockDevice{
				{
					File: "/dev/loop0",
					Partitions: []*model.Partition{
						{Name: "BOOT", Position: 1, Size: "10M", Type: "ef00"},
						{Name: "SWAP", Position: 2, Size: "16M", Type: "ef00"},
						{Name: "ROOT", Position: 3, Size: "60M", Type: "ef00"},
					},
				},
				{
					File: "/dev/loop1",
					Partitions: []*model.Partition{
						{Name: "BOOT", Position: 1, Size: "10M", Type: "ef00"},
						{Name: "SWAP", Position: 2, Size: "16M", Type: "ef00"},
						{Name: "ROOT", Position: 3, Size: "60M", Type: "ef00"},
					},
				},
			},
			RaidArrays: []*model.RaidArray{
				{
					Name:    "BOOT",
					Devices: []*model.BlockDevice{},
					Level:   "1",
				},
				{
					Name:    "ROOT",
					Devices: []*model.BlockDevice{},
					Level:   "1",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			var test_disks []struct {
				loop       *losetup.Device
				image_file string
			}

			// Prepare underlying block devices (losetup)
			for _, bd := range tc.BlockDevices {
				loopdev, image_file, err := prepareTestDisk(1024)
				if err != nil {
					t.Error(err)
				}

				t.Logf("Prepared test block device. device: %v, image_file: %v\n", loopdev, image_file)

				// Since we don't know the block device names ahead of time in this case
				bd.File = loopdev.Path()

				// We collect the disks for cleanup later
				test_disks = append(test_disks, struct {
					loop       *losetup.Device
					image_file string
				}{&loopdev, image_file})

				// Create partitions (see TestPartitionDisk)
				if out, err := createPartitions(bd, bd.Partitions); err != nil {
					t.Log(out)
					t.Error(err)
				}
			}

			// Create RAID Array(s)
			for _, r := range tc.RaidArrays {
				// Again, because we don't know the block devices ahead of time
				r.Devices = nil

				// Find block devices for partitions that make up the array
				for _, bd := range tc.BlockDevices {
					for _, p := range bd.Partitions {
						if p.Name == r.Name {
							r.Devices = append(r.Devices, p.BlockDevice)
						}
					}
				}

				// Apply the partition to the block device
				if err := r.Create("linuxsw"); err != nil {
					t.Error(err)
				}

				// Grab the current state of the RAID array(s) on the system and log
				mdstat, err := os.ReadFile("/proc/mdstat")
				if err != nil {
					t.Fatal(err)
				}
				t.Log(string(mdstat))
			}

			// Disable any active raid arrays
			for _, r := range tc.RaidArrays {
				if err := r.Disable("linuxsw"); err != nil {
					t.Error(err)
				}

				if err := r.Delete("linuxsw"); err != nil {
					t.Error(err)
				}
			}

			// Cleanup
			for _, l := range test_disks {
				if k_out, err := kpartxDel(l.loop.Path()); err != nil {
					t.Log(k_out)
					t.Fatal(err)
				}

				if err := cleanupTestDisk(l.image_file, l.loop); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestPartitionDisk(t *testing.T) {
	tests := []struct {
		testName   string
		diskSize   int64
		partitions []*model.Partition
	}{
		{
			"small-simple", 100, []*model.Partition{
				{Name: "BOOT", Position: 1, Size: "10M", Type: "ef00"},
				{Name: "SWAP", Position: 2, Size: "16M", Type: "ef00"},
				{Name: "ROOT", Position: 3, Size: "60M", Type: "ef00"},
			},
		},
		{
			"larger", 5000, []*model.Partition{
				{Name: "FIRST", Position: 1, Size: "+512M", Type: "ef00"},
				{Name: "SECOND", Position: 2, Size: "+512M", Type: "8300"},
				{Name: "THIRD", Position: 3, Size: "+2GB", Type: "8300"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.testName, func(t *testing.T) {
			loopdev, image_file, err := prepareTestDisk(tc.diskSize)
			if err != nil {
				t.Error(err)
			}

			t.Logf("Prepared test block device. device: %v, image_file: %v\n", loopdev, image_file)

			bd, err := model.NewBlockDevice(loopdev.Path())
			if err != nil {
				t.Error(err)
			}

			if out, err := createPartitions(bd, tc.partitions); err != nil {
				t.Log(out)
				t.Error(err)
			}

			// TODO(jwb) We should do something to actually validate that the partiton structure on disk matches our expectations.

			if k_out, err := kpartxDel(loopdev.Path()); err != nil {
				t.Log(k_out)
				t.Fatal(err)
			}

			if err := cleanupTestDisk(image_file, &loopdev); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestFormatPartition(t *testing.T) {
	tests := []struct {
		Name       string
		DiskSize   int64
		Partitions []*model.Partition
	}{
		{
			Name:     "simple",
			DiskSize: 128,
			Partitions: []*model.Partition{
				{Name: "BOOT", Position: 1, Size: "10M", Type: "ef00", FileSystem: "ext4", FileSystemOptions: []string{"-L", "ROOT"}},
				{Name: "SWAP", Position: 2, Size: "16M", Type: "ef00", FileSystem: "ext4", FileSystemOptions: []string{"-L", "ROOT"}},
				{Name: "ROOT", Position: 3, Size: "60M", Type: "ef00", FileSystem: "ext4", FileSystemOptions: []string{"-L", "ROOT"}},
			},
		},
	}

	for _, tc := range tests {
		loopdev, image_file, err := prepareTestDisk(tc.DiskSize)
		if err != nil {
			t.Error(err)
		}

		t.Logf("Prepared test block device. device: %v, image_file: %v\n", loopdev, image_file)

		bd, err := model.NewBlockDevice(loopdev.Path())

		if err != nil {
			t.Error(err)
		}

		if out, err := createPartitions(bd, tc.Partitions); err != nil {
			t.Log(out)
			t.Error(err)
		}

		for _, p := range tc.Partitions {
			if out, err := p.Format(); err != nil {
				t.Log(out)
				t.Error(err)
			}
		}

		// TODO(jwb) Check that the filesystem was created properly.

		if k_out, err := kpartxDel(loopdev.Path()); err != nil {
			t.Log(k_out)
			t.Fatal(err)
		}

		if err := cleanupTestDisk(image_file, &loopdev); err != nil {
			t.Fatal(err)
		}
	}
}
