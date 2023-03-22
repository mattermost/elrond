// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

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
	ringCreateCmd.Flags().String("installation-group-name", "", "The installation group name to register with the ring.")
	ringCreateCmd.Flags().Int("installation-group-soak-time", 0, "The installation group soak time.")
	ringCreateCmd.Flags().String("installation-group-provisioner-group-id", "", "The installation group provisioner group ID to associate.")

	ringCreateCmd.Flags().Int("soak-time", 7200, "The soak time to consider a ring release stable.")
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
	ringReleaseCmd.Flags().Bool("force", false, "When set to true a release is forced and soaking times are ignored.")
	ringReleaseCmd.Flags().Bool("all-rings", false, "Whether all rings should be released.")
	ringReleaseCmd.Flags().Bool("pause", false, "Whether to pause a release in progress.")
	ringReleaseCmd.Flags().Bool("resume", false, "Whether to resume a paused release.")
	ringReleaseCmd.Flags().Bool("cancel", false, "Whether to cancel a release.")
	ringReleaseCmd.Flags().StringArray("env-variable", []string{}, "Additional environment variables for the installation group release. Accepts multiple values, for example: '... --env-variable abc=123 --env-variable def=456'")

	ringReleaseGetCmd.Flags().String("release", "", "The id of the release to return info.")
	ringReleaseGetCmd.MarkFlagRequired("release") //nolint

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
	ringCmd.AddCommand(ringReleaseGetCmd)
	ringCmd.AddCommand(ringUpdateCmd)
	ringCmd.AddCommand(ringDeleteCmd)
	ringCmd.AddCommand(ringGetCmd)
	ringCmd.AddCommand(ringListCmd)
	ringCmd.AddCommand(ringInstallationGroupCmd)
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
		installationGroupName, _ := command.Flags().GetString("installation-group-name")
		installationGroupSoakTime, _ := command.Flags().GetInt("installation-group-soak-time")
		installationGroupProvisionerGroupID, _ := command.Flags().GetString("installation-group-provisioner-group-id")
		soakTime, _ := command.Flags().GetInt("soak-time")
		image, _ := command.Flags().GetString("image")
		version, _ := command.Flags().GetString("version")

		installationGroup := &model.InstallationGroup{
			Name:               installationGroupName,
			SoakTime:           installationGroupSoakTime,
			ProvisionerGroupID: installationGroupProvisionerGroupID,
		}

		request := &model.CreateRingRequest{
			Name:              name,
			Priority:          priority,
			InstallationGroup: installationGroup,
			SoakTime:          soakTime,
			Image:             image,
			Version:           version,
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
		force, _ := command.Flags().GetBool("force")
		releaseAllRings, _ := command.Flags().GetBool("all-rings")
		pauseRelease, _ := command.Flags().GetBool("pause")
		resumeRelease, _ := command.Flags().GetBool("resume")
		cancelRelease, _ := command.Flags().GetBool("cancel")
		envVariables, _ := command.Flags().GetStringArray("env-variable")

		request := &model.RingReleaseRequest{
			Image:        image,
			Version:      version,
			Force:        force,
			EnvVariables: strings.Join(envVariables, ","),
		}

		dryRun, _ := command.Flags().GetBool("dry-run")
		if dryRun {
			err := printJSON(request)
			if err != nil {
				return errors.Wrap(err, "failed to print API request")
			}

			return nil
		}

		if pauseRelease {
			err := client.PauseRelease()
			if err != nil {
				return errors.Wrap(err, "failed to pause all pending releases")
			}

			return nil
		}

		if resumeRelease {
			err := client.ResumeRelease()
			if err != nil {
				return errors.Wrap(err, "failed to resume all paused releases")
			}

			return nil
		}

		if cancelRelease {
			err := client.CancelRelease()
			if err != nil {
				return errors.Wrap(err, "failed to cancel all pending releases")
			}

			return nil
		}

		if releaseAllRings {
			rings, err := client.ReleaseAllRings(request)
			if err != nil {
				return errors.Wrap(err, "failed to achieve an all rings release")
			}
			if err = printJSON(rings); err != nil {
				return errors.Wrap(err, "failed to print release response for an all rings release")
			}
		} else {
			ring, err := client.ReleaseRing(ringID, request)
			if err != nil {
				return errors.Wrapf(err, "failed to release a ring %s", ringID)
			}
			if err = printJSON(ring); err != nil {
				return errors.Wrapf(err, "failed to print ring %s release response", ringID)
			}
		}

		return nil
	},
}

var ringReleaseGetCmd = &cobra.Command{
	Use:   "get-release",
	Short: "Get a particular ring release.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		if _, err := url.Parse(serverAddress); err != nil {
			return errors.Wrap(err, "provided server address not a valid address")
		}

		client := model.NewClient(serverAddress)

		releaseID, _ := command.Flags().GetString("release")
		ringRelease, err := client.GetRingRelease(releaseID)
		if err != nil {
			return errors.Wrapf(err, "failed to query ring release %s", releaseID)
		}
		if ringRelease == nil {
			return nil
		}

		if err = printJSON(ringRelease); err != nil {
			return errors.Wrapf(err, "failed to print ring release %s response", releaseID)
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
			table.SetRowLine(true)
			table.SetHeader([]string{"ID", "STATE", "NAME", "PRIORITY", "INSTALLATION GROUPS", "SOAK TIME", "REMAINING SOAK TIME", "ACTIVERELEASE", "DESIREDRELEASE", "FORCE", "RELEASE AT"})

			for _, ring := range rings {
				activeRelease, err := client.GetRingRelease(ring.ActiveReleaseID)
				if err != nil {
					return errors.Wrap(err, "failed to get active release for table output")
				}
				desiredRelease, err := client.GetRingRelease(ring.DesiredReleaseID)
				if err != nil {
					return errors.Wrap(err, "failed to get active release for table output")
				}

				var igs []string
				if len(ring.InstallationGroups) > 0 {
					for _, ig := range ring.InstallationGroups {
						igs = append(igs, fmt.Sprintf("Name: %s, State: %s, Soaking: %d, Provisioner Group: %s, ReleaseAt: %d", ig.Name, ig.State, ig.SoakTime, ig.ProvisionerGroupID, ig.ReleaseAt))
					}

				}
				var remainTime = int64(0)
				timePassed := ((time.Now().UnixNano() - ring.ReleaseAt) / int64(time.Second))
				if timePassed < int64(ring.SoakTime) {
					remainTime = int64(ring.SoakTime) - timePassed
				}
				table.Append([]string{
					ring.ID,
					ring.State,
					ring.Name,
					strconv.Itoa(ring.Priority),
					strings.Join(igs, "\n"),
					strconv.Itoa(ring.SoakTime),
					strconv.FormatInt(remainTime, 10),
					fmt.Sprintf("%s:%s", activeRelease.Image, activeRelease.Version),
					fmt.Sprintf("%s:%s", desiredRelease.Image, desiredRelease.Version),
					strconv.FormatBool(desiredRelease.Force),
					strconv.FormatInt(ring.ReleaseAt, 10),
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
