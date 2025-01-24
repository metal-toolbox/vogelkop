package cmd

import (
	"cmp"
	"context"
	"errors"
	"os"
	"strings"
	"time"

	common "github.com/metal-toolbox/bmc-common"
	"github.com/metal-toolbox/ironlib"
	"github.com/metal-toolbox/ironlib/actions"
	"github.com/metal-toolbox/ironlib/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// nolint:gocyclo // easier to read in one big function I think
func init() {
	cmd := &cobra.Command{
		Use:   "wipe /dev/disk",
		Short: "Wipes all data from a disk",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("requires at least one arg") // nolint:goerr113
			}

			_, err := os.Open(args[0])
			return err
		},
		Run: func(cmd *cobra.Command, args []string) {
			timeout, err := cmd.Flags().GetDuration("timeout")
			if err != nil {
				logger.With("error", err).Fatal("--timeout argument is invalid")
			}

			verbose, err := cmd.Flags().GetBool("debug")
			if err != nil {
				logger.With("error", err).Fatal("--debug argument is invalid")
			}

			driveName := args[0]

			logger := logrus.New()
			logger.Formatter = new(logrus.TextFormatter)
			if verbose {
				logger.SetLevel(logrus.TraceLevel)
			}
			l := logger.WithField("drive", driveName)

			ctx := cmp.Or(cmd.Context(), context.Background())
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			collector, err := ironlib.New(logger)
			if err != nil {
				l.WithError(err).Fatal("exiting")
			}

			inventory, err := collector.GetInventory(ctx, actions.WithDynamicCollection())
			if err != nil {
				l.WithError(err).Fatal("exiting")
			}

			var drive *common.Drive
			for _, d := range inventory.Drives {
				if d.LogicalName == driveName {
					drive = d
					break
				}
			}
			if drive == nil {
				l.Fatal("unable to find disk")
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
				}
			}

			if wiper == nil {
				l.WithFields(logrus.Fields{
					"capabilities": drive.Capabilities,
					"protocol":     drive.Protocol,
				}).Fatal("failed find appropriate drive wiper")
			}

			logger = logrus.New()
			logger.SetLevel(logrus.DebugLevel)
			err = wiper.WipeDrive(ctx, logger, drive)
			if err != nil {
				l.Fatal("failed to wipe drive")
			}
		},
	}

	diskCommand.PersistentFlags().Duration("timeout", 1*time.Minute, "Time to wait for wipe to complete")
	diskCommand.AddCommand(cmd)
}
