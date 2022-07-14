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
	ringInstallationGroupRegisterCmd.Flags().String("installation-group-name", "", "Additional installation group for the ring.")

	ringInstallationGroupRegisterCmd.Flags().String("ring", "", "The id of the ring to register the installation groups.")
	ringInstallationGroupRegisterCmd.Flags().String("provisioner-group-id", "", "The id of the provisioner group that will have 1to1 relationship with the elrond installation group.")
	ringInstallationGroupRegisterCmd.Flags().Int("soak-time", 0, "The soak time to consider an installation group release stable.")
	ringInstallationGroupRegisterCmd.MarkFlagRequired("ring")
	ringInstallationGroupRegisterCmd.MarkFlagRequired("installation-group-name")
	ringInstallationGroupRegisterCmd.MarkFlagRequired("provisioner-group-id")

	ringInstallationGroupUpdateCmd.Flags().String("installation-group", "", "The id of the installation group to update.")
	ringInstallationGroupUpdateCmd.Flags().String("name", "", "The name to set to the installation group.")
	ringInstallationGroupUpdateCmd.Flags().String("provisioner-group-id", "", "The id of the provisioner group that will have 1to1 relationship with the elrond installation group.")
	ringInstallationGroupUpdateCmd.Flags().Int("soak-time", 0, "The soak time to set to the installation group.")
	ringInstallationGroupUpdateCmd.MarkFlagRequired("installation-group")

	ringInstallationGroupDeleteCmd.Flags().String("installation-group", "", "ID of the installation group to be removed from the ring.")
	ringInstallationGroupDeleteCmd.Flags().String("ring", "", "The id of the ring from which installation group should be removed.")
	ringInstallationGroupDeleteCmd.MarkFlagRequired("ring")
	ringInstallationGroupDeleteCmd.MarkFlagRequired("installation-group")

	ringInstallationGroupCmd.AddCommand(ringInstallationGroupRegisterCmd)
	ringInstallationGroupCmd.AddCommand(ringInstallationGroupUpdateCmd)
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
		installationGroupName, _ := command.Flags().GetString("installation-group-name")
		soakTime, _ := command.Flags().GetInt("soak-time")
		provisionerGroupID, _ := command.Flags().GetString("provisioner-group-id")

		request := &model.RegisterInstallationGroupRequest{
			Name:               installationGroupName,
			SoakTime:           soakTime,
			ProvisionerGroupID: provisionerGroupID,
		}

		dryRun, _ := command.Flags().GetBool("dry-run")
		if dryRun {
			return runDryRun(request)
		}

		ring, err := client.RegisterRingInstallationGroup(ringID, request)
		if err != nil {
			return errors.Wrap(err, "failed to add ring installation group")
		}

		err = printJSON(ring)
		if err != nil {
			return errors.Wrap(err, "failed to print ring installation group response")
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

var ringInstallationGroupUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Updates installation group from the ring.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		installationGroupID, _ := command.Flags().GetString("installation-group")
		name, _ := command.Flags().GetString("name")
		soakTime, _ := command.Flags().GetInt("soak-time")
		provisionerGroupID, _ := command.Flags().GetString("provisioner-group-id")

		request := &model.UpdateInstallationGroupRequest{
			Name:               name,
			SoakTime:           soakTime,
			ProvisionerGroupID: provisionerGroupID,
		}

		dryRun, _ := command.Flags().GetBool("dry-run")
		if dryRun {
			err := printJSON(request)
			if err != nil {
				return errors.Wrap(err, "failed to print API request")
			}

			return nil
		}

		installationGroup, err := client.UpdateInstallationGroup(installationGroupID, request)
		if err != nil {
			return errors.Wrap(err, "failed to update installation group")
		}

		if err = printJSON(installationGroup); err != nil {
			return errors.Wrapf(err, "failed to print installation group %s response", request.Name)
		}

		return nil
	},
}

func runDryRun(request interface{}) error {
	err := printJSON(request)
	if err != nil {
		return errors.Wrap(err, "failed to print API request")
	}

	return nil
}
