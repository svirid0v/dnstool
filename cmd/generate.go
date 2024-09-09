package cmd

import (
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate configuration for lancache container(s)",
	Long:  `Generate and manipulate configuration files for lancache container(s)`,
}

func init() {
	generateCmd.AddCommand(lancacheDNSCmd)
}
