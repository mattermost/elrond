// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

// Package main is the entry point to the Mattermost Elrond server and CLI.
package main

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "elrond",
	Short: "Elrond is a tool to manage and support ring-based deployments in Mattermost Cloud.",
	Run: func(cmd *cobra.Command, args []string) {
		serverCmd.RunE(cmd, args) //nolint
	},
	// SilenceErrors allows us to explicitly log the error returned from rootCmd below.
	SilenceErrors: true,
}

func init() {
	rootCmd.MarkFlagRequired("database") //nolint

	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(ringCmd)
	rootCmd.AddCommand(schemaCmd)
	rootCmd.AddCommand(webhookCmd)
	rootCmd.AddCommand(securityCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.WithError(err).Error("command failed")
		os.Exit(1)
	}
}
