package elrond

import (
	"fmt"
	"time"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/mattermost/elrond/model"
	"github.com/pkg/errors"
)

func (provisioner *ElProvisioner) AddGrafanaAnnotations(text string, ring *model.Ring, installationGroup *model.InstallationGroup, release *model.RingRelease) error {
	if provisioner.params.GrafanaURL != "" && len(provisioner.params.GrafanaTokens) > 0 {
		logger := provisioner.logger.WithField("installationgroup", installationGroup.ID)
		logger.Infof("Adding Grafana release annotations for %s release", installationGroup.ID)
		for _, token := range provisioner.params.GrafanaTokens {
			grafanaClient, err := gapi.New(provisioner.params.GrafanaURL, gapi.Config{APIKey: token})
			if err != nil {
				return errors.Wrap(err, "failed to initiate Grafana client")
			}
			id, err := grafanaClient.NewAnnotation(&gapi.Annotation{
				Text: text,
				Tags: []string{fmt.Sprintf("ring:%s", ring.Name), fmt.Sprintf("installation-group:%s", installationGroup.ProvisionerGroupID), fmt.Sprintf("image:%s", release.Image), fmt.Sprintf("version:%s", release.Version), "elrond"},
				Time: time.Now().UnixNano() / int64(time.Millisecond),
			})
			if err != nil {
				return errors.Wrap(err, "failed to create Grafana annotation")
			}
			logger.Infof("Annotation created successfully with ID %d", id)
		}
	}
	return nil
}
