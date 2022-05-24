package cmd

import (
	"go.uber.org/zap"
	"github.com/spf13/cobra"

	"vogelkop/internal"
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
	defer logger.Sync()

	logger.Debugw("Logger configured.")
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