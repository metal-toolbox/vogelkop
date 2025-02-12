package cmd

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	diskfs "github.com/diskfs/go-diskfs"
	losetup "github.com/freddierice/go-losetup/v2"
	common "github.com/metal-toolbox/bmc-common"
	"github.com/metal-toolbox/ironlib/actions"
	ilmodel "github.com/metal-toolbox/ironlib/model"
	"github.com/metal-toolbox/vogelkop/internal/command"
	"github.com/sirupsen/logrus"
)

const testDiskSize = 128

var (
	ErrWriteFailed   = errors.New("failed to write bytes")
	ErrUnimplemented = errors.New("unimplemented")
)

type fakeCollector struct {
	fakeDrives []string
}

func (f *fakeCollector) GetInventory(_ context.Context, _ ...actions.Option) (*common.Device, error) {
	dev := &common.Device{}
	for _, driveName := range f.fakeDrives {
		dev.Drives = append(dev.Drives, &common.Drive{
			Common: common.Common{
				LogicalName:  driveName,
				Capabilities: []*common.Capability{{}},
			},
			Protocol:      "sata",
			CapacityBytes: testDiskSize * 1024 * 1024,
		})
	}
	return dev, nil
}

func (fakeCollector) ApplyUpdate(_ context.Context, _, _ string) error {
	return ErrUnimplemented
}

func (fakeCollector) InstallUpdates(_ context.Context, _ *ilmodel.UpdateOptions) error {
	return ErrUnimplemented
}

func (fakeCollector) SetBIOSConfiguration(_ context.Context, _ map[string]string) error {
	return ErrUnimplemented
}

func (fakeCollector) GetBIOSConfiguration(_ context.Context) (map[string]string, error) {
	return nil, ErrUnimplemented
}

func (fakeCollector) GetModel() string {
	return ""
}

func (fakeCollector) GetVendor() string {
	return ""
}

func (fakeCollector) RebootRequired() bool {
	return false
}

func (fakeCollector) UpdatesApplied() bool {
	return false
}

func (fakeCollector) GetInventoryOEM(_ context.Context, _ *common.Device, _ *ilmodel.UpdateOptions) error {
	return ErrUnimplemented
}

func (fakeCollector) ListAvailableUpdates(_ context.Context, _ *ilmodel.UpdateOptions) (*common.Device, error) {
	return nil, ErrUnimplemented
}

func (fakeCollector) UpdateRequirements(_ context.Context, _, _, _ string) (*ilmodel.UpdateRequirements, error) {
	return nil, ErrUnimplemented
}

func prepareTestDiskWithRandomFileName() (*losetup.Device, string, error) {
	randBytes := make([]byte, 16)
	_, _ = rand.Read(randBytes)
	diskImg := filepath.Join(os.TempDir(), hex.EncodeToString(randBytes))
	return prepareTestDisk(diskImg)
}

func prepareTestDisk(diskImg string) (*losetup.Device, string, error) {
	diskSize := int64(testDiskSize) * 1024 * 1024

	_, err := diskfs.Create(diskImg, diskSize, diskfs.Raw, 512)
	if err != nil {
		return nil, "", err
	}

	loopdev, err := losetup.Attach(diskImg, 0, false)
	if err != nil {
		return nil, "", err
	}

	return &loopdev, diskImg, nil
}

func cleanupTestDisk(imageName string, loopdev *losetup.Device) (err error) {
	if err = loopdev.Detach(); err != nil {
		return
	}

	if err = os.Remove(imageName); err != nil {
		return
	}

	return
}

func createContentAndVerifySuccess(loopdevPath string, content []byte) (*os.File, func() error, error) {
	loopdevFile, err := os.OpenFile(loopdevPath, os.O_RDWR, 0)
	if err != nil {
		return nil, nil, err
	}

	_, err = loopdevFile.Write(content)
	if err != nil {
		return nil, nil, err
	}
	_, err = loopdevFile.Seek(0, 0)
	if err != nil {
		return nil, nil, err
	}

	readBuffer := make([]byte, 10)
	_, err = loopdevFile.Read(readBuffer)
	if err != nil {
		return nil, nil, err
	}
	if !bytes.Equal(readBuffer, content) {
		return nil, nil, fmt.Errorf("%w: %v got %v, expect %v", ErrWriteFailed, loopdevFile, readBuffer, content)
	}

	return loopdevFile, loopdevFile.Close, nil
}

func verifyWipeSuccess(wipedDrives, unWipedDrives []*os.File, content []byte) error {
	emptyContent := make([]byte, 10)
	for _, loopdevFile := range wipedDrives {
		_, err := loopdevFile.Seek(0, 0)
		if err != nil {
			return err
		}
		readBuffer := make([]byte, 10)
		_, err = loopdevFile.Read(readBuffer)
		if err != nil {
			return err
		}
		if !bytes.Equal(readBuffer, emptyContent) {
			return fmt.Errorf("%w: %v got %v, expect %v", ErrWriteFailed, loopdevFile, readBuffer, content)
		}
	}
	for _, loopdevFile := range unWipedDrives {
		_, err := loopdevFile.Seek(0, 0)
		if err != nil {
			return err
		}
		readBuffer := make([]byte, 10)
		_, err = loopdevFile.Read(readBuffer)
		if err != nil {
			return err
		}
		if !bytes.Equal(readBuffer, content) {
			return fmt.Errorf("%w: %v got %v, expect %v", ErrWriteFailed, loopdevFile, readBuffer, content)
		}
	}
	return nil
}

func TestWiping(t *testing.T) {
	tests := []struct {
		desc      string
		diskCount int
		wipeCount int
	}{
		{
			desc:      "collector collects 0 drive",
			diskCount: 0,
			wipeCount: 0,
		},
		{
			desc:      "Wipe 0 drive",
			diskCount: 5,
			wipeCount: 0,
		},
		{
			desc:      "Wipe 1 drive from 1 total drive",
			diskCount: 1,
			wipeCount: 1,
		},
		{
			desc:      "Wipe 1 drive from 5 total drives",
			diskCount: 5,
			wipeCount: 1,
		},
		{
			desc:      "Wipe 3 drive from 5 total drives",
			diskCount: 5,
			wipeCount: 3,
		},
		{
			desc:      "Wipe all 5 drives",
			diskCount: 5,
			wipeCount: 5,
		},
		{
			desc:      "Wipe all 16 drives",
			diskCount: 16,
			wipeCount: 16,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// To avoid gocritic deferInLoop
			var cleanupFuncs []func()
			defer func() {
				for _, cleanup := range cleanupFuncs {
					cleanup()
				}
			}()

			collector := &fakeCollector{}
			fileContent := []byte{2, 3, 5, 7, 11, 13, 17, 19, 23, 29}
			var wipeDrives []string
			var wipedDrives, unWipedDrives []*os.File
			for i := range tc.diskCount {
				loopdev, imageFile, err := prepareTestDiskWithRandomFileName()
				if err != nil {
					t.Error(err)
					return
				}
				cleanupFuncs = append(cleanupFuncs, func() {
					if err = cleanupTestDisk(imageFile, loopdev); err != nil {
						t.Fatal(err)
					}
				})

				t.Logf("Prepared test block device. device: %v, image_file: %v\n", loopdev, imageFile)

				loopdevPath := loopdev.Path()
				_, err = command.Call(ctx, "kpartx", "-a", loopdevPath)
				if err != nil {
					t.Fatal(err)
				}
				collector.fakeDrives = append(collector.fakeDrives, loopdevPath)
				loopdevFile, cleanup, err := createContentAndVerifySuccess(loopdevPath, fileContent)
				if err != nil {
					t.Fatal((err))
				}
				cleanupFuncs = append(cleanupFuncs, func() {
					if err := cleanup(); err != nil {
						t.Fatal(err)
					}
				})

				if i < tc.wipeCount {
					wipeDrives = append(wipeDrives, loopdevPath)
					wipedDrives = append(wipedDrives, loopdevFile)
				} else {
					unWipedDrives = append(unWipedDrives, loopdevFile)
				}
			}

			logger := logrus.New()
			wipeDisks(ctx, wipeDrives, collector, logger, true)
			if err := verifyWipeSuccess(wipedDrives, unWipedDrives, fileContent); err != nil {
				t.Errorf("failed to wipe drives: %v", err)
			}
		})
	}
}
