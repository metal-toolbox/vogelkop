package model_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmc-toolbox/common"
	diskfs "github.com/diskfs/go-diskfs"
	losetup "github.com/freddierice/go-losetup/v2"
	"github.com/metal-toolbox/vogelkop/internal/command"
	"github.com/metal-toolbox/vogelkop/pkg/model"
)

func cleanupTestDisk(imageName string, loopdev *losetup.Device) (err error) {
	if err = loopdev.Detach(); err != nil {
		return
	}

	if err = os.Remove(imageName); err != nil {
		return
	}

	return
}

func prepareTestDisk(size int) (loopdev losetup.Device, diskImg string, err error) {
	diskImg = tempFileName("", "")

	diskSize := int64(size) * 1024 * 1024

	_, err = diskfs.Create(diskImg, diskSize, diskfs.Raw, 512)
	if err != nil {
		return
	}

	loopdev, err = losetup.Attach(diskImg, 0, false)
	if err != nil {
		return
	}

	return
}

func createPartitions(ctx context.Context, bd *model.BlockDevice, partitions []*model.Partition) (out string, err error) {
	for _, p := range partitions {
		p.BlockDevice = bd

		out, err = p.Create(ctx)
		if err != nil {
			return
		}
	}

	// Since we are using loopback devices for this test, we need to use kpartx
	// to make the partition devices accessible
	out, err = kpartxAdd(ctx, bd.File)
	if err != nil {
		return
	}

	for _, p := range partitions {
		var partitionBd *model.BlockDevice

		partitionBd, err = model.NewBlockDevice(p.GetLoopBlockDevice())
		if err != nil {
			return
		}

		p.BlockDevice = partitionBd
	}

	return out, err
}

func kpartxAdd(ctx context.Context, device string) (out string, err error) {
	out, err = command.Call(ctx, "kpartx", "-a", device)
	return
}

func kpartxDel(ctx context.Context, device string) (out string, err error) {
	out, err = command.Call(ctx, "kpartx", "-d", device)
	return
}

// tempFileName returns a 'random' filename with a given prefix and/or suffix.
func tempFileName(prefix, suffix string) string {
	randBytes := make([]byte, 16)
	_, _ = rand.Read(randBytes)

	return filepath.Join(os.TempDir(), prefix+hex.EncodeToString(randBytes)+suffix)
}

//nolint:gocyclo // Will look to extract some of this into reusable chunks at a later date
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
			ctx := context.Background()

			var testDisks []struct {
				loop      *losetup.Device
				imageFile string
			}

			// Prepare underlying block devices (losetup)
			for _, bd := range tc.BlockDevices {
				loopdev, imageFile, err := prepareTestDisk(1024)
				if err != nil {
					t.Error(err)
				}

				t.Logf("Prepared test block device. device: %v, image_file: %v\n", loopdev, imageFile)

				// Since we don't know the block device names ahead of time in this case
				bd.File = loopdev.Path()

				// We collect the disks for cleanup later
				testDisks = append(testDisks, struct {
					loop      *losetup.Device
					imageFile string
				}{&loopdev, imageFile})

				// Create partitions (see TestPartitionDisk)
				if out, err := createPartitions(ctx, bd, bd.Partitions); err != nil {
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
				if err := r.Create(ctx, common.SlugRAIDImplLinuxSoftware); err != nil {
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
				if out, err := r.Delete(ctx, common.SlugRAIDImplLinuxSoftware); err != nil {
					t.Log(out)
					t.Error(err)
				}
			}

			// Cleanup
			for _, l := range testDisks {
				if out, err := kpartxDel(ctx, l.loop.Path()); err != nil {
					t.Log(out)
					t.Fatal(err)
				}

				if err := cleanupTestDisk(l.imageFile, l.loop); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestPartitionDisk(t *testing.T) {
	tests := []struct {
		diskSize   int
		testName   string
		partitions []*model.Partition
	}{
		{
			testName: "small-simple", diskSize: 100, partitions: []*model.Partition{
				{Name: "BOOT", Position: 1, Size: "10M", Type: "ef00"},
				{Name: "SWAP", Position: 2, Size: "16M", Type: "ef00"},
				{Name: "ROOT", Position: 3, Size: "60M", Type: "ef00"},
			},
		},
		{
			testName: "larger", diskSize: 5000, partitions: []*model.Partition{
				{Name: "FIRST", Position: 1, Size: "+512M", Type: "ef00"},
				{Name: "SECOND", Position: 2, Size: "+512M", Type: "8300"},
				{Name: "THIRD", Position: 3, Size: "+2GB", Type: "8300"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.testName, func(t *testing.T) {
			ctx := context.Background()

			loopdev, imageFile, err := prepareTestDisk(tc.diskSize)
			if err != nil {
				t.Error(err)
			}

			t.Logf("Prepared test block device. device: %v, image_file: %v\n", loopdev, imageFile)

			bd, err := model.NewBlockDevice(loopdev.Path())
			if err != nil {
				t.Error(err)
			}

			if out, err := createPartitions(ctx, bd, tc.partitions); err != nil {
				t.Log(out)
				t.Error(err)
			}

			// TODO(jwb) We should do something to actually validate that the partiton structure on disk matches our expectations.

			if out, err := kpartxDel(ctx, loopdev.Path()); err != nil {
				t.Log(out)
				t.Fatal(err)
			}

			if err := cleanupTestDisk(imageFile, &loopdev); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestFormatPartition(t *testing.T) {
	tests := []struct {
		Name       string
		DiskSize   int
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
		ctx := context.Background()

		loopdev, imageFile, err := prepareTestDisk(tc.DiskSize)
		if err != nil {
			t.Error(err)
		}

		t.Logf("Prepared test block device. device: %v, image_file: %v\n", loopdev, imageFile)

		bd, err := model.NewBlockDevice(loopdev.Path())
		if err != nil {
			t.Error(err)
		}

		if out, err := createPartitions(ctx, bd, tc.Partitions); err != nil {
			t.Log(out)
			t.Error(err)
		}

		for _, p := range tc.Partitions {
			if out, err := p.Format(ctx); err != nil {
				t.Log(out)
				t.Error(err)
			}
		}

		// TODO(jwb) Check that the filesystem was created properly.

		if out, err := kpartxDel(ctx, loopdev.Path()); err != nil {
			t.Log(out)
			t.Fatal(err)
		}

		if err := cleanupTestDisk(imageFile, &loopdev); err != nil {
			t.Fatal(err)
		}
	}
}
