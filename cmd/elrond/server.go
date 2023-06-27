// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/elrond/internal/api"
	"github.com/mattermost/elrond/internal/elrond"
	"github.com/mattermost/elrond/internal/store"
	"github.com/mattermost/elrond/internal/supervisor"

	"github.com/mattermost/elrond/model"
	"github.com/pkg/errors"
	logrus "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	defaultLocalServerAPI = "http://localhost:3018"
)

var instanceID string

func init() {
	instanceID = model.NewID()

	// General
	serverCmd.PersistentFlags().String("database", "sqlite://elrond.db", "The database backing the elrond server.")
	serverCmd.PersistentFlags().String("listen", ":3018", "The interface and port on which to listen.")
	serverCmd.PersistentFlags().Bool("debug", false, "Whether to output debug logs.")
	serverCmd.PersistentFlags().Bool("machine-readable-logs", false, "Output the logs in machine readable format.")
	serverCmd.PersistentFlags().String("provisioner-server", "http://localhost:8075", "The provisioning server whose API will be queried.")
	serverCmd.PersistentFlags().Int("provisioner-group-release-timeout", 3600, "The provisioner group release timeout")
	serverCmd.PersistentFlags().String("grafana-token", "", "The Grafana token for the Grafana intergration.")
	serverCmd.PersistentFlags().StringSlice("grafana-orgs", []string{""}, "The Grafana Orgs to integrate with")

	// Supervisors
	serverCmd.PersistentFlags().Int("poll", 30, "The interval in seconds to poll for background work.")
	serverCmd.PersistentFlags().Bool("ring-supervisor", true, "Whether this server will run a ring supervisor or not.")
	serverCmd.PersistentFlags().Bool("installationgroup-supervisor", true, "Whether this server will run an installation group supervisor or not.")
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run the elrond server.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		debug, _ := command.Flags().GetBool("debug")
		if debug {
			logger.SetLevel(logrus.DebugLevel)
		}

		machineLogs, _ := command.Flags().GetBool("machine-readable-logs")
		if machineLogs {
			logger.SetFormatter(&logrus.JSONFormatter{})
		}

		provisionerServer, _ := command.Flags().GetString("provisioner-server")

		provisionerGroupReleaseTimeout, _ := command.Flags().GetInt("provisioner-group-release-timeout")

		logger := logger.WithField("instance", instanceID)

		sqlStore, err := sqlStore(command)
		if err != nil {
			return err
		}

		currentVersion, err := sqlStore.GetCurrentVersion()
		if err != nil {
			return err
		}
		serverVersion := store.LatestVersion()

		// Require the schema to be at least the server version, and also the same major
		// version.
		if currentVersion.LT(serverVersion) || currentVersion.Major != serverVersion.Major {
			return errors.Errorf("server requires at least schema %s, current is %s", serverVersion, currentVersion)
		}

		ringSupervisor, _ := command.Flags().GetBool("ring-supervisor")
		installationGroupSupervisor, _ := command.Flags().GetBool("installationgroup-supervisor")
		if !ringSupervisor && !installationGroupSupervisor {
			logger.Warn("Server will be running with no supervisors. Only API functionality will work.")
		}

		wd, err := os.Getwd()
		if err != nil {
			wd = "error getting working directory"
			logger.WithError(err).Error("Unable to get current working directory")
		}

		logger.WithFields(logrus.Fields{
			"build-hash":                   model.BuildHash,
			"ring-supervisor":              ringSupervisor,
			"installationgroup-supervisor": installationGroupSupervisor,
			"store-version":                currentVersion,
			"working-directory":            wd,
		}).Info("Starting Mattermost Elrond Server")

		deprecationWarnings(logger, command)

		provisioningParams := elrond.ProvisioningParams{
			ProvisionerGroupReleaseTimeout: provisionerGroupReleaseTimeout,
		}

		// Setup the provisioner.
		elrondProvisioner := elrond.NewElrondProvisioner(
			provisioningParams,
			logger,
			provisionerServer,
		)

		var multiDoer supervisor.MultiDoer
		if ringSupervisor {
			multiDoer = append(multiDoer, supervisor.NewRingSupervisor(sqlStore, elrondProvisioner, instanceID, logger))
		}
		if installationGroupSupervisor {
			multiDoer = append(multiDoer, supervisor.NewInstallationGroupSupervisor(sqlStore, elrondProvisioner, instanceID, logger))
		}

		// Setup the supervisor to effect any requested changes. It is wrapped in a
		// scheduler to trigger it periodically in addition to being poked by the API
		// layer.
		poll, _ := command.Flags().GetInt("poll")
		if poll == 0 {
			logger.WithField("poll", poll).Info("Scheduler is disabled")
		}

		supervisor := supervisor.NewScheduler(multiDoer, time.Duration(poll)*time.Second)
		defer supervisor.Close()

		router := mux.NewRouter()

		api.Register(router, &api.Context{
			Store:             sqlStore,
			Supervisor:        supervisor,
			Elrond:            elrondProvisioner,
			Logger:            logger,
			ProvisionerServer: provisionerServer,
		})

		listen, _ := command.Flags().GetString("listen")
		srv := &http.Server{
			Addr:           listen,
			Handler:        router,
			ReadTimeout:    180 * time.Second,
			WriteTimeout:   180 * time.Second,
			IdleTimeout:    time.Second * 180,
			MaxHeaderBytes: 1 << 20,
			ErrorLog:       log.New(&logrusWriter{logger}, "", 0),
		}

		go func() {
			logger.WithField("addr", srv.Addr).Info("Listening")
			err := srv.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				logger.WithError(err).Error("Failed to listen and serve")
			}
		}()

		c := make(chan os.Signal, 1)
		// We'll accept graceful shutdowns when quit via:
		//  - SIGINT (Ctrl+C)
		//  - SIGTERM (Ctrl+/) (Kubernetes pod rolling termination)
		// SIGKILL and SIGQUIT will not be caught.
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)

		// Block until we receive a valid signal.
		sig := <-c
		logger.WithField("shutdown-signal", sig.String()).Info("Shutting down")

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		srv.Shutdown(ctx) //nolint

		return nil
	},
}

// deprecationWarnings performs all checks for deprecated settings and warns if
// any are found.
func deprecationWarnings(logger logrus.FieldLogger, cmd *cobra.Command) {
	// Add deprecation logic here.
}
