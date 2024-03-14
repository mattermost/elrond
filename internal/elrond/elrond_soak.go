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
	if err != nil {
		return errors.Wrap(err, "failed to query thanos")
	}

	for _, result := range results {
		if len(result) > 0 {
			for _, rawMetric := range result {
				sloService, exists := rawMetric.Metric["slo_service"]
				if !exists {
					continue
				}
				return errors.Errorf("SLO service %s is showing a high API error rate. Soaking failed, stopping release", sloService)
			}
		} else {
			logger.Info("SLO metric checks looking good...")
		}
	}

	return nil
}

func querySLOMetrics(ring *model.Ring, url string, queryTime time.Time, logger *logrus.Entry) ([]pmodel.Vector, error) {
	client, err := api.NewClient(api.Config{Address: url})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create prometheus client")
	}

	v1api := v1.NewAPI(client)
	var results []pmodel.Vector

	for _, installationGroup := range ring.InstallationGroups {
		query := fmt.Sprintf("((slo:sli_error:ratio_rate5m{slo_service='%[1]s-ring-%[2]s'} > (14.4 * 0.005)) and ignoring(slo_window)(slo:sli_error:ratio_rate1h{slo_service='%[1]s-ring-%[2]s'} > (14.4 * 0.005))) or ignoring(slo_window)((slo:sli_error:ratio_rate30m{slo_service='%[1]s-ring-%[2]s'} > (6 * 0.005)) and ignoring(slo_window)(slo:sli_error:ratio_rate6h{slo_service='%[1]s-ring-%[2]s'} > (3.3 * 0.005))) or vector(0)", installationGroup.Name, installationGroup.ProvisionerGroupID)
		var lastErr error
		// Retry mechanism for Thanos network connectivity issues.
		for attempt := 0; attempt < 10; attempt++ {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			logger.Infof("Running Thanos query %s, attempt %d", query, attempt+1)
			result, warnings, err := v1api.Query(ctx, query, queryTime)
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					logger.Errorf("Query failed due to timeout: %v", err)
				} else {
					logger.Errorf("Query failed due to an error: %v", err)
				}
			}
			cancel()

			if err == nil {
				if len(warnings) > 0 {
					logger.Warnf("Encountered warnings obtaining metrics: %s", strings.Join(warnings, ", "))
				}
				results = append(results, result.(pmodel.Vector))
				break
			}

			lastErr = err
			logger.Warnf("Query failed: %v", err)
			if attempt+1 < 10 {
				time.Sleep(time.Second * time.Duration(2<<attempt)) // Exponential backoff
			}
		}

		if lastErr != nil {
			return nil, errors.Wrap(lastErr, "failed to query after retries")
		}
	}

	return results, nil
}
