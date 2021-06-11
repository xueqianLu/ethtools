package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	cobra.EnableCommandSorting = false
}

type command struct {
	root *cobra.Command
}

type option func(*command)

func newCommand(opts ...option) (c *command, err error) {
	c = &command{
		root: &cobra.Command{
			Use:           "ethtool",
			Short:         "Ethereum tools",
			SilenceErrors: true,
			SilenceUsage:  true,
		},
	}

	for _, o := range opts {
		o(c)
	}

	if err := c.initTxCmd(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *command) Execute() (err error) {
	return c.root.Execute()
}

// Execute parses command line arguments and runs appropriate functions.
func Execute() (err error) {
	c, err := newCommand()
	if err != nil {
		return err
	}
	return c.Execute()
}
