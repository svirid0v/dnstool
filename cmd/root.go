package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dnstool",
	Short: "dnstool is a utility to generate configuration for the lancache-dns container",
	Long: `A replacement utility for the configuration generator bash script:
https://github.com/lancachenet/lancache-dns/blob/d626a74c02c7a8383eeaaab493fcdffe536aea95/overlay/hooks/entrypoint-pre.d/10_generate_config.sh
utilised to generate configuration for lancache-dns containers`,
}

func init() {
	rootCmd.AddCommand(generateCmd)
}

func Execute() error {
	return rootCmd.Execute()
}
