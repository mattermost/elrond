// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"github.com/mattermost/elrond/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	ringInstallationGroupRegisterCmd.Flags().StringArray("installation-group", []string{}, "Additional installation groups for the ring. Accepts multiple values, for example: '... --installation-group group-123 --installation-group group-1234'")

	ringInstallationGroupRegisterCmd.Flags().String("ring", "", "The id of the ring to register the installation groups.")
	ringInstallationGroupRegisterCmd.MarkFlagRequired("ring")
	ringInstallationGroupRegisterCmd.MarkFlagRequired("installation-group")

	ringInstallationGroupDeleteCmd.Flags().String("installation-group", "", "Name of the installation group to be removed from the ring.")
	ringInstallationGroupDeleteCmd.Flags().String("ring", "", "The id of the ring from which installation group should be removed.")
	ringInstallationGroupDeleteCmd.MarkFlagRequired("ring")
	ringInstallationGroupDeleteCmd.MarkFlagRequired("installation-group")

	ringInstallationGroupCmd.AddCommand(ringInstallationGroupRegisterCmd)
	ringInstallationGroupCmd.AddCommand(ringInstallationGroupDeleteCmd)
}

var ringInstallationGroupCmd = &cobra.Command{
	Use:   "installation-group",
	Short: "Manipulate installation groups of rings managed by the elrond server.",
}

var ringInstallationGroupRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Registers installation groups to the ring.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		ringID, _ := command.Flags().GetString("ring")
		installationGroups, _ := command.Flags().GetStringArray("installation-group")

		request := newAddInstallationGroupsRequest(installationGroups)

		dryRun, _ := command.Flags().GetBool("dry-run")
		if dryRun {
			return runDryRun(request)
		}

		ring, err := client.RegisterRingInstallationGroups(ringID, request)
		if err != nil {
			return errors.Wrap(err, "failed to add ring installation groups")
		}

		err = printJSON(ring)
		if err != nil {
			return errors.Wrap(err, "failed to print ring installation groups response")
		}

		return nil
	},
}

var ringInstallationGroupDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Deletes installation group from the ring.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		RingID, _ := command.Flags().GetString("ring")
		installationGroup, _ := command.Flags().GetString("installation-group")

		err := client.DeleteRingInstallationGroup(RingID, installationGroup)
		if err != nil {
			return errors.Wrap(err, "failed to delete ring installation group")
		}

		return nil
	},
}

func newAddInstallationGroupsRequest(installationGroups []string) *model.RegisterInstallationGroupsRequest {
	return &model.RegisterInstallationGroupsRequest{
		InstallationGroups: installationGroups,
	}
}

func runDryRun(request interface{}) error {
	err := printJSON(request)
	if err != nil {
		return errors.Wrap(err, "failed to print API request")
	}

	return nil
}
