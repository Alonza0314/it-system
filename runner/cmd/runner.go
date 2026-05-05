package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/Alonza0314/it-system/runner/config"
	"github.com/Alonza0314/it-system/runner/internal"
	"github.com/Alonza0314/it-system/runner/logger"

	loggergoUtil "github.com/Alonza0314/logger-go/v2/util"
	"github.com/free-ran-ue/util"
	"github.com/spf13/cobra"
)

var runnerCmd = &cobra.Command{
	Use: "runner",
	Run: runnerFunc,
}

func init() {
	runnerCmd.Flags().StringP("config", "c", "config.yaml", "Path to the configuration file")
	if err := runnerCmd.MarkFlagRequired("config"); err != nil {
		panic(err)
	}
}

func runnerFunc(cmd *cobra.Command, args []string) {
	runnerConfigFilePath, err := cmd.Flags().GetString("config")
	if err != nil {
		panic(err)
	}

	runnerConfig := config.Config{}
	if err := util.LoadFromYaml(runnerConfigFilePath, &runnerConfig); err != nil {
		panic(err)
	}

	logger := logger.NewRunnerLogger(loggergoUtil.LogLevelString(runnerConfig.Logger.Level), "", true)

	tokenBytes, err := os.ReadFile(runnerConfig.Runner.TokenPath)
	if err != nil {
		logger.RunLog.Errorf("Failed to read token file: %v", err)
		return
	}

	runner := internal.NewRunner(&runnerConfig, string(tokenBytes), logger)
	if runner == nil {
		panic("failed to initialize the runner")
	}

	runner.Start()
	defer runner.Stop()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
}

func Execute() {
	if err := runnerCmd.Execute(); err != nil {
		panic(err)
	}
}
