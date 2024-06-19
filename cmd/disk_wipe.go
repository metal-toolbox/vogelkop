package cmd

import (
	"cmp"
	"context"
	"errors"
	"os"
	"time"

	"github.com/bmc-toolbox/common"
	"github.com/metal-toolbox/ironlib"
	"github.com/metal-toolbox/ironlib/actions"
	"github.com/metal-toolbox/ironlib/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

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
			// nolint:gocritic // will have more cases soon, remove nolint then
			switch drive.Protocol {
			case "nvme":
				wiper = utils.NewNvmeCmd(verbose)
			}

			if wiper == nil {
				l.Fatal("failed find appropriate wiper drive")
			}

			err = wiper.WipeDrive(ctx, logger, drive)
			if err != nil {
				l.Fatal("failed to wipe drive")
			}
		},
	}

	diskCommand.PersistentFlags().Duration("timeout", 1*time.Minute, "Time to wait for wipe to complete")
	diskCommand.AddCommand(cmd)
}
