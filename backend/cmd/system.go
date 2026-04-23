package cmd

import (
	"backend/config"
	"backend/internal"
	"backend/logger"
	"os"
	"os/signal"
	"syscall"

	loggergoUtil "github.com/Alonza0314/logger-go/v2/util"
	"github.com/free-ran-ue/util"
	"github.com/spf13/cobra"
)

var systemCmd = &cobra.Command{
	Use: "system",
	Run: systemFunc,
}

func init() {
	systemCmd.Flags().StringP("config", "c", "config.yaml", "Path to the configuration file")
	if err := systemCmd.MarkFlagRequired("config"); err != nil {
		panic(err)
	}
}

func systemFunc(cmd *cobra.Command, args []string) {
	systemConfigFilePath, err := cmd.Flags().GetString("config")
	if err != nil {
		panic(err)
	}

	systemConfig := config.Config{}
	if err := util.LoadFromYaml(systemConfigFilePath, &systemConfig); err != nil {
		panic(err)
	}

	logger := logger.NewBackendLogger(loggergoUtil.LogLevelString(systemConfig.Logger.Level), "", true)

	system := internal.NewBackend(&systemConfig, logger)
	if system == nil {
		panic("failed to initialize the backend")
	}

	system.Start()
	defer system.Stop()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
}

func Execute() {
	if err := systemCmd.Execute(); err != nil {
		panic(err)
	}
}
