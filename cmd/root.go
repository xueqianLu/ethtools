package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	. "github.com/xueqianLu/ethtools/utils"
	"os"
)

var logLevel string

func init() {
	cobra.OnInitialize(initConfig)
	cobra.EnableCommandSorting = false

	rootCmd.AddCommand(chainPareCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.PersistentFlags().StringVar(&logLevel, "loglevel", "info", "log level")

	chainPareCmd.Flags().String(Chain1Flag, "", "the first chain")
	chainPareCmd.Flags().String(Chain2Flag, "", "the second chain")
	chainPareCmd.Flags().String(AccountFileFlag, "accounts.json", "the account file")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	InitLog(logLevel, "")

	viper.AutomaticEnv() // read in environment variables that match
	//viper.SetConfigName("config") // name of config file (without extension)
	//viper.AddConfigPath(".")
	//// If a config file is found, read it in.
	//if err := viper.ReadInConfig(); err == nil {
	//	//log.Info("Using config file:", viper.ConfigFileUsed())
	//} else {
	//	log.WithField("error", err).Warn("Read config failed")
	//	return
	//}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Errorf("Program execute error: %s", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "ethtools",
	Short: "A set of ethereum tools",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}
