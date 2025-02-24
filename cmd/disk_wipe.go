package cmd

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	common "github.com/metal-toolbox/bmc-common"
	"github.com/metal-toolbox/ironlib"
	"github.com/metal-toolbox/ironlib/actions"
	"github.com/metal-toolbox/ironlib/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	ErrDriveNotExist      = errors.New("drive does not exist")
	ErrDriveWiperNotFound = errors.New("failed to find appropriate drive wiper")
)

type diskWipeResult struct {
	DriveName string `json:"drive_name"`
	Fail      bool   `json:"success"`
	Err       string `json:"error,omitempty"`
}

func wipeDisks(ctx context.Context, drivesName []string, collector actions.DeviceManager, logger *logrus.Logger, verbose bool) {
	inventory, err := collector.GetInventory(ctx, actions.WithDynamicCollection())
	if err != nil {
		logger.WithError(err).Fatal("exiting")
	}

	wipeResultsCh := make(chan *diskWipeResult, 1)
	go func() {
		var failureResults []*diskWipeResult
		for result := range wipeResultsCh {
			if result.Fail {
				failureResults = append(failureResults, result)
			}
		}
		if len(failureResults) > 0 {
			jsonData, marshalErr := json.Marshal(failureResults)
			if marshalErr != nil {
				logger.Fatalf("Error marshaling %v to JSON: %v", failureResults, marshalErr)
				return
			}
			logger.Fatal(string(jsonData))
		}
	}()

	var wg sync.WaitGroup
	wg.Add(len(drivesName))
	for _, driveName := range drivesName {
		go func() {
			l := logger.WithField("drive", driveName)
			err = wipeOneDisk(ctx, inventory, driveName, &wg, verbose)
			wipeResultsCh <- &diskWipeResult{driveName, true, err.Error()}
			if err != nil {
				// we may want to see error message as soon as possible
				l.Errorf("failed to wipe disk %v: error %v", driveName, err)
				return
			}
			l.Infof("wipe drive %v done", driveName)
		}()
	}
	wg.Wait()
	close(wipeResultsCh)
}

// nolint:gocyclo // easier to read in one big function I think
func wipeOneDisk(ctx context.Context, inventory *common.Device, driveName string, wg *sync.WaitGroup, verbose bool) error {
	defer wg.Done()

	var drive *common.Drive
	for _, d := range inventory.Drives {
		if d.LogicalName == driveName {
			drive = d
			break
		}
	}
	if drive == nil {
		return ErrDriveNotExist
	}

	// Pick the most appropriate wipe based on the disk type and/or features supported
	var wiper actions.DriveWiper
	switch drive.Protocol {
	case "nvme":
		wiper = utils.NewNvmeCmd(verbose)
	case "sata", "sas":
		// Lets figure out the drive capabilities in an easier format
		var sanitize bool
		var esee bool
		var trim bool
		for _, cap := range drive.Capabilities {
			switch {
			case cap.Description == "encryption supports enhanced erase":
				esee = cap.Enabled
			case cap.Description == "SANITIZE feature":
				sanitize = cap.Enabled
			case strings.HasPrefix(cap.Description, "Data Set Management TRIM supported"):
				trim = cap.Enabled
			}
		}

		switch {
		case sanitize || esee:
			// Drive supports Sanitize or Enhanced Erase, so we use hdparm
			wiper = utils.NewHdparmCmd(verbose)
		case trim:
			// Drive supports TRIM, so we use blkdiscard
			wiper = utils.NewBlkdiscardCmd(verbose)
		default:
			// Drive does not support any preferred wipe method so we fall back to filling it up with zeros
			wiper = utils.NewFillZeroCmd(verbose)
		}
	}

	if wiper == nil {
		return fmt.Errorf("capabilities: %v, protocol: %v: %w", drive.Capabilities, drive.Protocol, ErrDriveWiperNotFound)
	}

	wiperLogger := logrus.New()
	wiperLogger.SetLevel(logrus.DebugLevel)
	if err := wiper.WipeDrive(ctx, wiperLogger, drive); err != nil {
		return fmt.Errorf("wiper.WipeDrive() failed to wipe drive: %w", err)
	}
	return nil
}

// nolint:gocyclo // easier to read in one big function I think
func init() {
	cmd := &cobra.Command{
		Use:   "wipe /dev/disk,/dev/diska,...",
		Short: "Wipes all data from disks(comma-separated)",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("requires at least one arg") // nolint:goerr113
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			timeout, err := cmd.Flags().GetDuration("timeout")
			if err != nil {
				logger.With("error", err).Fatal("--timeout argument is invalid")
			}

			if timeout <= 0 {
				logger.With("error", err).Fatal("--timeout should be positive")
			}

			verbose, err := cmd.Flags().GetBool("debug")
			if err != nil {
				logger.With("error", err).Fatal("--debug argument is invalid")
			}

			logger := logrus.New()
			logger.Formatter = new(logrus.TextFormatter)
			if verbose {
				logger.SetLevel(logrus.TraceLevel)
			}

			ctx := cmp.Or(cmd.Context(), context.Background())
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			var drivesName []string
			drivesNameMap := make(map[string]struct{})
			// is regex better?
			for _, driveName := range strings.Split(args[0], ",") {
				_, err = os.Stat(driveName)
				if err != nil {
					// should we ignore errors and let inventory collector to handle errors like permission, I/O, os errors
					// or handle file not exist error here is good enough?
					logger.Warnf("invalid drive %v: %v", driveName, err)
					continue
				}
				if _, exists := drivesNameMap[driveName]; exists {
					logger.Warnf("duplicate drive input %v", driveName)
					continue
				}
				drivesNameMap[driveName] = struct{}{}
				drivesName = append(drivesName, driveName)
			}

			collector, err := ironlib.New(logger)
			if err != nil {
				logger.WithError(err).Fatal("exiting")
			}

			wipeDisks(ctx, drivesName, collector, logger, verbose)
		},
	}

	diskCommand.PersistentFlags().Duration("timeout", 1*time.Minute, "Time to wait for wipe to complete")
	diskCommand.AddCommand(cmd)
}
