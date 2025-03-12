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

type wiperInfo struct {
	Disk        string `json:"disk"`
	Action      string `json:"action"`
	Method      string `json:"method"`
	ElapsedTime int    `json:"elapsed_time"`
	Result      string `json:"result"`
}

func wipeDisks(ctx context.Context, drivesName []string, collector actions.DeviceManager, wipeResults []*wiperInfo, logger *logrus.Logger, logResultFilename string, verbose bool) {
	inventory, err := collector.GetInventory(ctx, actions.WithDynamicCollection())
	if err != nil {
		logger.WithError(err).Fatal("exiting")
	}

	var hasFailure bool
	var wg sync.WaitGroup
	wg.Add(len(drivesName))
	for _, driveName := range drivesName {
		go func() {
			defer wg.Done()
			l := logger.WithField("drive", driveName)
			wi := &wiperInfo{
				Disk:   driveName,
				Result: "success",
			}
			startTime := time.Now()
			if err = wipeOneDisk(ctx, inventory, wi, verbose); err != nil {
				hasFailure = true
				wi.Result = "failure"
				l.Errorf("failed to wipe disk %v: error %v", driveName, err)
			} else {
				l.Infof("wipe drive %v done", driveName)
			}
			wi.ElapsedTime = int(time.Since(startTime).Round(time.Second).Seconds())
			wipeResults = append(wipeResults, wi)
		}()
	}
	wg.Wait()
	wipeResultsJSON, marshalErr := json.MarshalIndent(wipeResults, "", "  ") // pretty printing
	if marshalErr != nil {
		logger.Fatalf("Error marshaling %v to JSON: %v", wipeResults, marshalErr)
	}

	if logResultFilename != "" {
		file, err := os.Create(logResultFilename)
		if err != nil {
			logger.Fatalf("failed to create %v: %v", logResultFilename, err)
		}
		defer file.Close()

		if _, err := file.Write(wipeResultsJSON); err != nil {
			logger.Fatalf("failed to write result to %v: %v", logResultFilename, err)
		}

		if hasFailure {
			logger.Fatal(string(wipeResultsJSON))
		}
	}
	logger.Info(string(wipeResultsJSON))
}

// nolint:gocyclo // easier to read in one big function I think
func wipeOneDisk(ctx context.Context, inventory *common.Device, wi *wiperInfo, verbose bool) error {
	var drive *common.Drive
	for _, d := range inventory.Drives {
		if d.LogicalName == wi.Disk {
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
		var ber bool
		var cer bool
		var cese bool
		for _, cap := range drive.Capabilities {
			switch cap.Name {
			case "ber":
				ber = cap.Enabled
			case "cer":
				cer = cap.Enabled
			case "cese":
				cese = cap.Enabled
			}
		}
		switch {
		case cer:
			wi.Method = "sanitize"
			wi.Action = "CryptoErase"
		case ber:
			wi.Method = "sanitize"
			wi.Action = "BlockErase"
		case cese:
			wi.Method = "format"
			wi.Action = "CryptographicErase"
		default:
			wi.Method = "format"
			wi.Action = "UserDataErase"
		}
		wiper = utils.NewNvmeCmd(verbose)
	case "sata", "sas":
		// Lets figure out the drive capabilities in an easier format
		var sanitize bool
		var esee bool
		var trim bool
		var eseu bool
		var bee bool
		var cse bool
		for _, cap := range drive.Capabilities {
			switch {
			case cap.Description == "encryption supports enhanced erase":
				esee = cap.Enabled
			case cap.Description == "SANITIZE feature":
				sanitize = cap.Enabled
			case strings.HasPrefix(cap.Description, "Data Set Management TRIM supported"):
				trim = cap.Enabled
			case cap.Description == "BLOCK ERASE EXT":
				bee = cap.Enabled
			case cap.Description == "CRYPTO SCRAMBLE EXT":
				cse = cap.Enabled
			case strings.HasPrefix(cap.Description, "erase time:"):
				eseu = strings.Contains(cap.Description, "enhanced")
			}
		}

		switch {
		case sanitize || esee:
			// It is better if ironlib util can export an API to provide cap info, or
			// WipeDrive can return methods/actions it uses:
			// https://github.com/metal-toolbox/ironlib/blob/main/utils/hdparm.go#L217-L237
			switch {
			case sanitize && cse:
				wi.Method = "sanitize"
				wi.Action = "sanitize-crypto-scramble"
			case sanitize && bee:
				wi.Method = "sanitize"
				wi.Action = "sanitize-block-erase"
			case esee && eseu:
				wi.Method = "security-erase-enhanced"
			}
			// Drive supports Sanitize or Enhanced Erase, so we use hdparm
			wiper = utils.NewHdparmCmd(verbose)
		case trim:
			// Drive supports TRIM, so we use blkdiscard
			wi.Method = "blkdiscard"
			wiper = utils.NewBlkdiscardCmd(verbose)
		default:
			// Drive does not support any preferred wipe method so we fall back to filling it up with zeros
			wi.Method = "fillzero"
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

			logResultFilename, err := cmd.Flags().GetString("output")
			if err != nil {
				logger.With("error", err).Fatal("--output argument is invalid")
			}

			logger := logrus.New()
			logger.Formatter = new(logrus.TextFormatter)
			if verbose {
				logger.SetLevel(logrus.TraceLevel)
			}

			ctx := cmp.Or(cmd.Context(), context.Background())
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			var wipeResults []*wiperInfo
			var drivesName []string
			drivesNameMap := make(map[string]struct{})
			// is regex better?
			for _, driveName := range args {
				_, err = os.Stat(driveName)
				if err != nil {
					// should we ignore errors and let inventory collector to handle errors like permission, I/O, os errors
					// or handle file not exist error here is good enough?
					logger.Warnf("invalid drive %v: %v", driveName, err)
					wipeResults = append(wipeResults, &wiperInfo{
						Disk:   driveName,
						Result: "failure",
					})
					continue
				}
				if _, exists := drivesNameMap[driveName]; exists {
					logger.Warnf("duplicate drive input %v", driveName)
					wipeResults = append(wipeResults, &wiperInfo{
						Disk:   driveName,
						Result: "failure",
					})
					continue
				}
				drivesNameMap[driveName] = struct{}{}
				drivesName = append(drivesName, driveName)
			}

			collector, err := ironlib.New(logger)
			if err != nil {
				logger.WithError(err).Fatal("exiting")
			}

			wipeDisks(ctx, drivesName, collector, wipeResults, logger, logResultFilename, verbose)
		},
	}

	diskCommand.PersistentFlags().String("output", "", "log wiping results to the file with json format")
	diskCommand.PersistentFlags().Duration("timeout", 1*time.Minute, "Time to wait for wipe to complete")
	diskCommand.AddCommand(cmd)
}
