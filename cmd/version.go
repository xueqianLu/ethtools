package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/xueqianLu/ethtools/versions"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of ethtools",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(versions.DetailVersion())
	},
}
