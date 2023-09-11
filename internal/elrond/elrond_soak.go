package elrond

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mattermost/elrond/model"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	pmodel "github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
)

func checkSLOs(ring *model.Ring, thanosURL string, logger *logrus.Entry) error {
	results, err := querySLOMetrics(ring, thanosURL, time.Now(), logger)
	for _, result := range results {
		if len(result) > 0 {
			for _, rawMetric := range result {
				sloService, ok := rawMetric.Metric["slo_service"]
				if !ok {
					continue
				}
				return errors.Errorf("SLO service %s is showing a high API error rate. Soaking failed, stopping release", sloService)
			}
		} else {
			logger.Info("SLO metric checks looking good...")
		}
	}

	if err != nil {
		return errors.Wrap(err, "failed to query thanos")
	}
	return nil
}

func querySLOMetrics(ring *model.Ring, url string, queryTime time.Time, logger *logrus.Entry) ([]pmodel.Vector, error) {
	client, err := api.NewClient(api.Config{Address: url})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create prometheus client")
	}

	v1api := v1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var results []pmodel.Vector
	for _, installationGroup := range ring.InstallationGroups {
		query := fmt.Sprintf("((slo:sli_error:ratio_rate5m{slo_service='%[1]s-ring-%[2]s'} > (14.4 * 0.005)) and ignoring(slo_window)(slo:sli_error:ratio_rate1h{slo_service='%[1]s-ring-%[2]s'} > (14.4 * 0.005))) or ignoring(slo_window)((slo:sli_error:ratio_rate30m{slo_service='%[1]s-ring-%[2]s'} > (6 * 0.005)) and ignoring(slo_window)(slo:sli_error:ratio_rate6h{slo_service='%[1]s-ring-%[2]s'} > (3.3 * 0.005)))", installationGroup.Name, installationGroup.ProvisionerGroupID)
		result, warnings, err := v1api.Query(ctx, query, queryTime)
		if err != nil {
			return nil, errors.Wrap(err, "failed to query")
		}

		if len(warnings) > 0 {
			return nil, errors.Errorf("encounted warnings obtaining metrics: %s", strings.Join(warnings, ", "))
		}
		results = append(results, result.(pmodel.Vector))
	}

	return results, nil
}
