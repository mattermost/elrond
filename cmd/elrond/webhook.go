// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"net/url"
	"os"

	"github.com/mattermost/elrond/model"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	webhookCmd.PersistentFlags().String("server", defaultLocalServerAPI, "The elrond server whose API will be queried.")

	webhookCreateCmd.Flags().String("owner", "", "An opaque identifier describing the owner of the webhook.")
	webhookCreateCmd.Flags().String("url", "", "The callback URL of the webhook.")
	webhookCreateCmd.MarkFlagRequired("owner") //nolint
	webhookCreateCmd.MarkFlagRequired("url")   //nolint

	webhookGetCmd.Flags().String("webhook", "", "The id of the webhook to be fetched.")
	webhookGetCmd.MarkFlagRequired("webhook") //nolint

	webhookListCmd.Flags().String("owner", "", "The owner by which to filter webhooks.")
	webhookListCmd.Flags().Int("page", 0, "The page of webhooks to fetch, starting at 0.")
	webhookListCmd.Flags().Int("per-page", 100, "The number of webhooks to fetch per page.")
	webhookListCmd.Flags().Bool("include-deleted", false, "Whether to include deleted webhooks.")
	webhookListCmd.Flags().Bool("table", false, "Whether to display the returned webhook list in a table or not")

	webhookDeleteCmd.Flags().String("webhook", "", "The id of the webhook to be deleted.")
	webhookDeleteCmd.MarkFlagRequired("webhook") //nolint

	webhookCmd.AddCommand(webhookCreateCmd)
	webhookCmd.AddCommand(webhookGetCmd)
	webhookCmd.AddCommand(webhookListCmd)
	webhookCmd.AddCommand(webhookDeleteCmd)
}

var webhookCmd = &cobra.Command{
	Use:   "webhook",
	Short: "Manipulate webhooks managed by the elrond server.",
}

var webhookCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a webhook.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		if _, err := url.Parse(serverAddress); err != nil {
			return errors.Wrap(err, "provided server address not a valid address")
		}

		client := model.NewClient(serverAddress)

		ownerID, _ := command.Flags().GetString("owner")
		url, _ := command.Flags().GetString("url")

		webhook, err := client.CreateWebhook(&model.CreateWebhookRequest{
			OwnerID: ownerID,
			URL:     url,
		})
		if err != nil {
			return errors.Wrap(err, "failed to create webhook")
		}

		if err = printJSON(webhook); err != nil {
			return err
		}

		return nil
	},
}

var webhookGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a particular webhook.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		if _, err := url.Parse(serverAddress); err != nil {
			return errors.Wrap(err, "provided server address not a valid address")
		}

		client := model.NewClient(serverAddress)

		webhookID, err := command.Flags().GetString("webhook")
		if err != nil {
			return errors.Wrap(err, "failed to read webhookid")
		}

		webhook, err := client.GetWebhook(webhookID)
		if err != nil {
			return errors.Wrap(err, "failed to query webhook")
		}
		if webhook == nil {
			return nil
		}

		if err = printJSON(webhook); err != nil {
			return err
		}

		return nil
	},
}

var webhookListCmd = &cobra.Command{
	Use:   "list",
	Short: "List created webhooks.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		if _, err := url.Parse(serverAddress); err != nil {
			return errors.Wrap(err, "provided server address not a valid address")
		}

		client := model.NewClient(serverAddress)

		owner, _ := command.Flags().GetString("owner")
		page, _ := command.Flags().GetInt("page")
		perPage, _ := command.Flags().GetInt("per-page")
		includeDeleted, _ := command.Flags().GetBool("include-deleted")
		webhooks, err := client.GetWebhooks(&model.GetWebhooksRequest{
			OwnerID:        owner,
			Page:           page,
			PerPage:        perPage,
			IncludeDeleted: includeDeleted,
		})
		if err != nil {
			return errors.Wrap(err, "failed to query webhooks")
		}

		outputToTable, _ := command.Flags().GetBool("table")
		if outputToTable {
			table := tablewriter.NewWriter(os.Stdout)
			table.SetAlignment(tablewriter.ALIGN_LEFT)
			table.SetHeader([]string{"ID", "OWNER", "URL"})

			for _, webhook := range webhooks {
				table.Append([]string{webhook.ID, webhook.OwnerID, webhook.URL})
			}
			table.Render()

			return nil
		}

		if err = printJSON(webhooks); err != nil {
			return err
		}

		return nil
	},
}

var webhookDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a webhook.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		if _, err := url.Parse(serverAddress); err != nil {
			return errors.Wrap(err, "provided server address not a valid address")
		}

		client := model.NewClient(serverAddress)

		webhookID, err := command.Flags().GetString("webhook")
		if err != nil {
			return errors.Wrap(err, "failed to read webhookid")
		}

		err = client.DeleteWebhook(webhookID)
		if err != nil {
			return errors.Wrap(err, "failed to delete webhook")
		}

		return nil
	},
}
