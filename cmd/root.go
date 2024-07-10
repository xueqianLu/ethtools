package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

func init() {
	cobra.EnableCommandSorting = false
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
