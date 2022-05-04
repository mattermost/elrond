// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"net/url"

	"github.com/mattermost/elrond/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	securityCmd.PersistentFlags().String("server", defaultLocalServerAPI, "The elrond server whose API will be queried.")

	securityRingCmd.PersistentFlags().String("ring", "", "The id of the ring.")
	securityRingCmd.MarkPersistentFlagRequired("ring") //nolint

	securityCmd.AddCommand(securityRingCmd)
	securityRingCmd.AddCommand(securityRingLockAPICmd)
	securityRingCmd.AddCommand(securityRingUnlockAPICmd)
}

var securityCmd = &cobra.Command{
	Use:   "security",
	Short: "Manage security locks for different elrond resources.",
}

var securityRingCmd = &cobra.Command{
	Use:   "ring",
	Short: "Manage security locks for ring resources.",
}

var securityRingLockAPICmd = &cobra.Command{
	Use:   "api-lock",
	Short: "Lock API changes on a given ring",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		if _, err := url.Parse(serverAddress); err != nil {
			return errors.Wrap(err, "provided server address not a valid address")
		}

		client := model.NewClient(serverAddress)

		ringID, _ := command.Flags().GetString("ring")
		err := client.LockAPIForRing(ringID)
		if err != nil {
			return errors.Wrap(err, "failed to lock ring API")
		}

		return nil
	},
}

var securityRingUnlockAPICmd = &cobra.Command{
	Use:   "api-unlock",
	Short: "Unlock API changes on a given ring",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		if _, err := url.Parse(serverAddress); err != nil {
			return errors.Wrap(err, "provided server address not a valid address")
		}

		client := model.NewClient(serverAddress)

		ringID, _ := command.Flags().GetString("ring")
		err := client.UnlockAPIForRing(ringID)
		if err != nil {
			return errors.Wrap(err, "failed to unlock ring API")
		}

		return nil
	},
}
