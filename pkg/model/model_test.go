package model_test

import (
	"os"
	"path/filepath"
	"testing"

	diskfs "github.com/diskfs/go-diskfs"
	losetup "github.com/freddierice/go-losetup/v2"
	"github.com/metal-toolbox/vogelkop/pkg/model"
)

func prepareTestDisk(image_name string, size int64) (device string, loopdev losetup.Device, disk_img string, err error) {
	disk_img = filepath.Join(os.TempDir(), image_name)

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

	device = loopdev.Path()
	return
}

// func TestConfigureRaid(t *testing.T) {
// 	tests := []struct {
// 		testName   string
// 		diskSize   int64
// 		partitions []model.Partition
// 	}{
// 		{
// 			"test1", 100, []model.Partition{
// 				{Name: "BOOT", Position: 1, Size: "10M", Type: "ef00"},
// 				{Name: "SWAP", Position: 2, Size: "16M", Type: "ef00"},
// 				{Name: "ROOT", Position: 3, Size: "60M", Type: "ef00"},
// 			},
// 		},
// 		{
// 			"test2", 5000, []model.Partition{
// 				{Name: "FIRST", Position: 1, Size: "+512M", Type: "ef00"},
// 				{Name: "SECOND", Position: 2, Size: "+512M", Type: "8300"},
// 				{Name: "THIRD", Position: 3, Size: "+2GB", Type: "8300"},
// 			},
// 		},
// 	}

// 	for _, tc := range tests {
// 		t.Run(tc.testName, func(t *testing.T) {
// 			testDevice, loopdev, disk_img, err := prepareTestDisk(job_name + "-" + tc.testName+".img", tc.diskSize)
// 			if err != nil {
// 				t.Error(err.Error())
// 			}

// 			for _, p := range tc.partitions {


// }

func TestPartitionDisk(t *testing.T) {
	job_name := t.Name()

	tests := []struct {
		testName   string
		diskSize   int64
		partitions []model.Partition
	}{
		{
			"test1", 100, []model.Partition{
				{Name: "BOOT", Position: 1, Size: "10M", Type: "ef00"},
				{Name: "SWAP", Position: 2, Size: "16M", Type: "ef00"},
				{Name: "ROOT", Position: 3, Size: "60M", Type: "ef00"},
			},
		},
		{
			"test2", 5000, []model.Partition{
				{Name: "FIRST", Position: 1, Size: "+512M", Type: "ef00"},
				{Name: "SECOND", Position: 2, Size: "+512M", Type: "8300"},
				{Name: "THIRD", Position: 3, Size: "+2GB", Type: "8300"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.testName, func(t *testing.T) {
			testDevice, loopdev, disk_img, err := prepareTestDisk(job_name + "-" + tc.testName+".img", tc.diskSize)
			if err != nil {
				t.Error(err.Error())
			}

			for _, p := range tc.partitions {
				p.BlockDevice, err = model.NewBlockDevice(testDevice)
				if err != nil {
					t.Error(err.Error())
				}

				t.Logf("p: %v\n", p)

				create_out, err := p.Create()
				if err != nil {
					t.Log(create_out)
					t.Error(err.Error())
				}
			}

			// TODO(jwb) We should do something to actually validate that the
			// partiton structure on disk matches our expectations.

			// Cleanup
			loopdev.Detach()
			os.Remove(disk_img)
		})
	}
}
