package cmd

import (
	"path/filepath"
	"strconv"
	"strings"

	version "github.com/metal-toolbox/vogelkop/internal"
	"github.com/metal-toolbox/vogelkop/pkg/model"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	logger  *zap.SugaredLogger
	rootCmd = &cobra.Command{
		Version: version.Version(),
		Use:     version.Name(),
		Short:   "Storage Management",
		Long:    "Configures storage from controller to filesystem",
	}
)

func init() {
	cobra.OnInitialize(initLogging)
	rootCmd.PersistentFlags().Bool("debug", false, "Debug Mode")
}

func initLogging() {
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)

	if b, err := rootCmd.Flags().GetBool("debug"); err == nil && b {
		cfg = zap.NewDevelopmentConfig()
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	}

	l, err := cfg.Build()
	if err != nil {
		panic(err)
	}

	logger = l.Sugar().With("app", version.Name(), "version", version.Version())

	defer loggerSync()

	logger.Debugw("Logger configured.")
}

func loggerSync() {
	if err := logger.Sync(); err != nil {
		logger.Debugw("logger failed to sync", "err", err)
	}
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func GetString(cmd *cobra.Command, key string) (v string) {
	v, err := cmd.Flags().GetString(key)
	if err != nil {
		logger.Panicw("Error processing "+key+" parameter.", "error", err)
	}

	return
}

func GetUint(cmd *cobra.Command, key string) (v uint) {
	v, err := cmd.Flags().GetUint(key)
	if err != nil {
		logger.Panicw("Error processing "+key+" parameter.", "error", err)
	}

	return
}

func GetStringSlice(cmd *cobra.Command, key string) (v []string) {
	v, err := cmd.Flags().GetStringSlice(key)
	if err != nil {
		logger.Panicw("Error processing "+key+" parameter.", "error", err)
	}

	return
}

func GetBool(cmd *cobra.Command, key string) (v bool) {
	v, err := cmd.Flags().GetBool(key)
	if err != nil {
		logger.Panicw("Error processing "+key+" parameter.", "error", err)
	}

	return
}

func getPartitionBlockDevice(device string, partition *model.Partition) (systemDevice string) {
	position := strconv.FormatInt(int64(partition.Position), 10)

	if strings.Contains(device, "loop") {
		systemDevice = getLoopPartitionBlockDevice(device, partition)
	} else {
		systemDevice = device + position
	}

	return
}

func getLoopPartitionBlockDevice(device string, partition *model.Partition) (systemDevice string) {
	position := strconv.FormatInt(int64(partition.Position), 10)
	deviceFile := filepath.Base(device)
	systemDevice = "/dev/mapper/" + deviceFile + "p" + position

	return
}

func markFlagAsRequired(cmd *cobra.Command, flagName string) {
	if err := cmd.MarkPersistentFlagRequired(flagName); err != nil {
		logger.Panicw("failed to mark flag as persistent", "err", err)
	}
}
