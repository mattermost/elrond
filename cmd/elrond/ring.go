// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"encoding/json"
	"net/url"
	"os"
	"strconv"

	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/mattermost/elrond/model"
)

func init() {
	ringCmd.PersistentFlags().String("server", defaultLocalServerAPI, "The ring server whose API will be queried.")
	ringCmd.PersistentFlags().Bool("dry-run", false, "When set to true, only print the API request without sending it.")

	ringCreateCmd.Flags().String("name", "", "The name that identifies the deployment ring.")
	ringCreateCmd.Flags().Int("priority", 1, "The priority of a new deployment ring.")
	ringCreateCmd.Flags().StringArray("installation-group", []string{}, "The installation group IDs to register with the ring. Accepts multiple values, for example: '... --installation-group abc --installation-group def'")

	ringCreateCmd.Flags().Int("soak-time", 7200, "The soak time to consider a release stable.")
	ringCreateCmd.Flags().String("image", "", "The Mattermost image to associate with this release ring.")
	ringCreateCmd.Flags().String("version", "", "The Mattermost version to associate with this release ring.")

	ringCreateCmd.MarkFlagRequired("priority") //nolint

	ringUpdateCmd.Flags().String("ring", "", "The id of the ring to update.")
	ringUpdateCmd.Flags().String("name", "", "The name to set to the deployment ring.")
	ringUpdateCmd.Flags().Int("priority", 0, "The priority to set to the deployment ring.")
	ringUpdateCmd.Flags().Int("soak-time", 0, "The soak time to set to the deployment ring.")
	ringUpdateCmd.Flags().String("image", "", "The Mattermost image to set to the deployment ring. This will not force a release.")
	ringUpdateCmd.Flags().String("version", "", "The Mattermost version to set to the deployment ring. This will not force a release.")

	ringUpdateCmd.MarkFlagRequired("ring") //nolint

	ringReleaseCmd.Flags().String("ring", "", "The id of the ring to be released.")
	ringReleaseCmd.Flags().String("image", "", "The Mattermost image to release to.")
	ringReleaseCmd.Flags().String("version", "", "The Mattermost version to release to.")

	ringReleaseCmd.MarkFlagRequired("ring") //nolint

	ringDeleteCmd.Flags().String("ring", "", "The id of the ring to be deleted.")
	ringDeleteCmd.MarkFlagRequired("ring") //nolint

	ringGetCmd.Flags().String("ring", "", "The id of the ring to be fetched.")
	ringGetCmd.MarkFlagRequired("ring") //nolint

	ringListCmd.Flags().Int("page", 0, "The page of rings to fetch, starting at 0.")
	ringListCmd.Flags().Int("per-page", 100, "The number of rings to fetch per page.")
	ringListCmd.Flags().Bool("include-deleted", false, "Whether to include deleted rings.")
	ringListCmd.Flags().Bool("table", false, "Whether to display the returned ring list in a table or not")

	ringCmd.AddCommand(ringCreateCmd)
	ringCmd.AddCommand(ringReleaseCmd)
	ringCmd.AddCommand(ringUpdateCmd)
	ringCmd.AddCommand(ringDeleteCmd)
	ringCmd.AddCommand(ringGetCmd)
	ringCmd.AddCommand(ringListCmd)
}

var ringCmd = &cobra.Command{
	Use:   "ring",
	Short: "Manipulate rings managed by the elrond server.",
}

func printJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "    ")
	return encoder.Encode(data)
}

var ringCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a ring.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		if _, err := url.Parse(serverAddress); err != nil {
			return errors.Wrap(err, "provided server address not a valid address")
		}

		client := model.NewClient(serverAddress)

		name, _ := command.Flags().GetString("name")
		priority, _ := command.Flags().GetInt("priority")
		installationGroups, _ := command.Flags().GetStringArray("installation-group")
		soakTime, _ := command.Flags().GetInt("soak-time")
		image, _ := command.Flags().GetString("image")
		version, _ := command.Flags().GetString("version")

		request := &model.CreateRingRequest{
			Name:               name,
			Priority:           priority,
			InstallationGroups: installationGroups,
			SoakTime:           soakTime,
			Image:              image,
			Version:            version,
		}

		dryRun, _ := command.Flags().GetBool("dry-run")
		if dryRun {
			err := printJSON(request)
			if err != nil {
				return errors.Wrap(err, "failed to print API request")
			}

			return nil
		}

		ring, err := client.CreateRing(request)
		if err != nil {
			return errors.Wrapf(err, "failed to create ring %s", request.Name)
		}

		if err = printJSON(ring); err != nil {
			return errors.Wrapf(err, "failed to print ring %s response", request.Name)
		}

		return nil
	},
}

var ringUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a ring.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		if _, err := url.Parse(serverAddress); err != nil {
			return errors.Wrap(err, "provided server address not a valid address")
		}

		client := model.NewClient(serverAddress)

		ringID, _ := command.Flags().GetString("ring")
		name, _ := command.Flags().GetString("name")
		priority, _ := command.Flags().GetInt("priority")
		soakTime, _ := command.Flags().GetInt("soak-time")
		image, _ := command.Flags().GetString("image")
		version, _ := command.Flags().GetString("version")

		request := &model.UpdateRingRequest{
			Name:     name,
			Priority: priority,
			SoakTime: soakTime,
			Image:    image,
			Version:  version,
		}

		dryRun, _ := command.Flags().GetBool("dry-run")
		if dryRun {
			err := printJSON(request)
			if err != nil {
				return errors.Wrap(err, "failed to print API request")
			}

			return nil
		}

		ring, err := client.UpdateRing(ringID, request)
		if err != nil {
			return errors.Wrapf(err, "failed to update ring %s", request.Name)
		}

		if err = printJSON(ring); err != nil {
			return errors.Wrapf(err, "failed to print ring %s response", request.Name)
		}

		return nil
	},
}

var ringReleaseCmd = &cobra.Command{
	Use:   "release",
	Short: "Release an elrond ring.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		if _, err := url.Parse(serverAddress); err != nil {
			return errors.Wrap(err, "provided server address not a valid address")
		}

		client := model.NewClient(serverAddress)
		ringID, _ := command.Flags().GetString("ring")
		image, _ := command.Flags().GetString("image")
		version, _ := command.Flags().GetString("version")

		request := &model.ReleaseRingRequest{
			Image:   image,
			Version: version,
		}

		dryRun, _ := command.Flags().GetBool("dry-run")
		if dryRun {
			err := printJSON(request)
			if err != nil {
				return errors.Wrap(err, "failed to print API request")
			}

			return nil
		}

		ring, err := client.ReleaseRing(ringID, request)
		if err != nil {
			return errors.Wrapf(err, "failed to release a ring %s", ringID)
		}

		if err = printJSON(ring); err != nil {
			return errors.Wrapf(err, "failed to print ring %s release response", ringID)
		}

		return nil
	},
}

var ringDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a ring.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		if _, err := url.Parse(serverAddress); err != nil {
			return errors.Wrap(err, "provided server address not a valid address")
		}

		client := model.NewClient(serverAddress)

		ringID, _ := command.Flags().GetString("ring")

		err := client.DeleteRing(ringID)
		if err != nil {
			return errors.Wrapf(err, "failed to delete ring %s", ringID)
		}

		return nil
	},
}

var ringGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a particular ring.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		if _, err := url.Parse(serverAddress); err != nil {
			return errors.Wrap(err, "provided server address not a valid address")
		}

		client := model.NewClient(serverAddress)

		ringID, _ := command.Flags().GetString("ring")
		ring, err := client.GetRing(ringID)
		if err != nil {
			return errors.Wrapf(err, "failed to query ring %s", ringID)
		}
		if ring == nil {
			return nil
		}

		if err = printJSON(ring); err != nil {
			return errors.Wrapf(err, "failed to print ring %s response", ringID)
		}

		return nil
	},
}

var ringListCmd = &cobra.Command{
	Use:   "list",
	Short: "List created rings.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		if _, err := url.Parse(serverAddress); err != nil {
			return errors.Wrap(err, "provided server address not a valid address")
		}

		client := model.NewClient(serverAddress)

		page, _ := command.Flags().GetInt("page")
		perPage, _ := command.Flags().GetInt("per-page")
		includeDeleted, _ := command.Flags().GetBool("include-deleted")
		rings, err := client.GetRings(&model.GetRingsRequest{
			Page:           page,
			PerPage:        perPage,
			IncludeDeleted: includeDeleted,
		})
		if err != nil {
			return errors.Wrap(err, "failed to query rings")
		}

		outputToTable, _ := command.Flags().GetBool("table")
		if outputToTable {
			table := tablewriter.NewWriter(os.Stdout)
			table.SetAlignment(tablewriter.ALIGN_LEFT)
			table.SetHeader([]string{"ID", "STATE", "NAME", "PRIORITY", "SOAK TIME", "IMAGE", "VERSION"})

			for _, ring := range rings {
				table.Append([]string{
					ring.ID,
					ring.State,
					ring.Name,
					strconv.Itoa(ring.Priority),
					strconv.Itoa(ring.SoakTime),
					ring.Image,
					ring.Version,
				})
			}
			table.Render()

			return nil
		}

		if err = printJSON(rings); err != nil {
			return errors.Wrap(err, "failed to print ring list response")
		}

		return nil
	},
}
