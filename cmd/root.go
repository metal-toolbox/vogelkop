package cmd

import (
	"go.uber.org/zap"
	"github.com/spf13/cobra"

	"github.com/metal-toolbox/vogelkop/internal"

	"strconv"
	"os/exec"
	"strings"
	"path/filepath"
)

var (
	logger *zap.SugaredLogger
	rootCmd = &cobra.Command{
		Version: version.Version(),
		Use:   version.Name(),
		Short: "Storage Management",
		Long:  "Configures storage from controller to filesystem",
	}
)

func init() {
	cobra.OnInitialize(initLogging)
	rootCmd.PersistentFlags().Bool("debug", false, "Debug Mode")
	rootCmd.PersistentFlags().String("log-level", "INFO", "Logging Level")
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

func GetString(cmd *cobra.Command, key string) (v string){
	v, err := cmd.Flags().GetString(key)
	if err != nil {
		logger.Panicw("Error processing " + key + " parameter.", "error", err)
	}

	return
}

func GetUint(cmd *cobra.Command, key string) (v uint){
	v, err := cmd.Flags().GetUint(key)
	if err != nil {
		logger.Panicw("Error processing " + key + " parameter.", "error", err)
	}

	return
}

func GetStringSlice(cmd *cobra.Command, key string) (v []string) {
	v, err := cmd.Flags().GetStringSlice(key)
	if err != nil {
		logger.Panicw("Error processing " + key + " parameter.", "error", err)
	}

	return
}

func GetBool(cmd *cobra.Command, key string) (v bool) {
	v, err := cmd.Flags().GetBool(key)
	if err != nil {
		logger.Panicw("Error processing " + key + " parameter.", "error", err)
	}

	return
}

func callCommand(cmd_name string, cmd_options ...string) (out []byte, err error) {
	cmd := exec.Command(cmd_name, cmd_options...)
	logger.Infow("running command", "cmd", cmd)
	out, err = cmd.CombinedOutput()

	if err != nil {
		logger.Debugf("%s\n", out)
		logger.Fatalw("Failed to run command",
			"cmd", cmd, "err", err, "out", string(out))
	}

	logger.Infow("command exited successfully", "cmd", cmd, "out", string(out))

	return
}

func getPartitionBlockDevice(device string, partition Partition) (system_device string) {
	position := strconv.FormatInt(int64(partition.Position),10)

	if strings.Contains(device, "loop") {
		system_device = getLoopPartitionBlockDevice(device, partition)
	} else {
		system_device = device + position
	}

	return
}

func getLoopPartitionBlockDevice(device string, partition Partition) (system_device string) {
	position := strconv.FormatInt(int64(partition.Position),10)
	device_file := filepath.Base(device)
	system_device = "/dev/mapper/" + device_file + "p" + position
	return
}

func markFlagAsRequired(cmd *cobra.Command, flag_name string) {
	if err := cmd.MarkPersistentFlagRequired(flag_name); err != nil {
		logger.Panicw("failed to mark flag as persistent", "err", err)
	}
}