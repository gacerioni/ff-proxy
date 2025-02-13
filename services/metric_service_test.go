package services

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/harness/ff-proxy/log"

	"github.com/harness/ff-proxy/domain"
	clientgen "github.com/harness/ff-proxy/gen/client"
	"github.com/stretchr/testify/assert"
)

const (
	defaultAccount    = "account"
	defaultMetricsURL = "https://events.ff.harness.io/api/1.0"
	defaultToken      = "token"
)

func (m mockService) PostMetricsWithResponse(ctx context.Context, environment clientgen.EnvironmentPathParam, params *clientgen.PostMetricsParams, body clientgen.PostMetricsJSONRequestBody, reqEditors ...clientgen.RequestEditorFn) (*clientgen.PostMetricsResponse, error) {
	var err error
	var resp *clientgen.PostMetricsResponse
	if m.postMetricsWithResp != nil {
		resp, err = m.postMetricsWithResp(environment)
	}

	return resp, err
}

var env123MetricsFlag1 = domain.MetricsRequest{
	EnvironmentID: "123",
	Metrics: clientgen.Metrics{
		TargetData: &[]clientgen.TargetData{
			{
				Identifier: "targetID",
				Name:       "target name",
				Attributes: []clientgen.KeyValue{{Key: "targetkey", Value: "targetvalue"}},
			},
		},
		MetricsData: &[]clientgen.MetricsData{
			{
				Timestamp:   int64(1234),
				Count:       1,
				MetricsType: "FFMETRICS",
				Attributes:  []clientgen.KeyValue{{Key: "featureIdentifier", Value: "flag1"}},
			},
		},
	},
}

var env123MetricsFlag2 = domain.MetricsRequest{
	EnvironmentID: "123",
	Metrics: clientgen.Metrics{
		TargetData: &[]clientgen.TargetData{},
		MetricsData: &[]clientgen.MetricsData{
			{
				Timestamp:   int64(5678),
				Count:       2,
				MetricsType: "FFMETRICS",
				Attributes:  []clientgen.KeyValue{{Key: "featureIdentifier", Value: "flag2"}},
			},
		},
	},
}

var env456MetricsFlag1 = domain.MetricsRequest{
	EnvironmentID: "456",
	Metrics: clientgen.Metrics{
		TargetData: &[]clientgen.TargetData{},
		MetricsData: &[]clientgen.MetricsData{
			{
				Timestamp:   int64(2345),
				Count:       1,
				MetricsType: "FFMETRICS",
				Attributes:  []clientgen.KeyValue{{Key: "featureIdentifier", Value: "flag1"}},
			},
		},
	},
}

func TestMetricService_StoreMetrics(t *testing.T) {

	testCases := map[string]struct {
		metrics  []domain.MetricsRequest
		enabled  bool
		expected map[string]domain.MetricsRequest
	}{
		"Given I save one environments metrics": {
			metrics:  []domain.MetricsRequest{env123MetricsFlag1},
			enabled:  true,
			expected: map[string]domain.MetricsRequest{"123": env123MetricsFlag1},
		},
		"Given I save two sets of metrics for one environment we combine them": {
			metrics: []domain.MetricsRequest{env123MetricsFlag1, env123MetricsFlag2},
			enabled: true,
			expected: map[string]domain.MetricsRequest{"123": {
				EnvironmentID: "123",
				Metrics: clientgen.Metrics{
					TargetData:  &[]clientgen.TargetData{(*env123MetricsFlag1.TargetData)[0]},
					MetricsData: &[]clientgen.MetricsData{(*env123MetricsFlag1.MetricsData)[0], (*env123MetricsFlag2.MetricsData)[0]},
				},
			}},
		},
		"Given I save two sets of metrics for different environments": {
			metrics: []domain.MetricsRequest{env123MetricsFlag1, env123MetricsFlag2, env456MetricsFlag1},
			enabled: true,
			expected: map[string]domain.MetricsRequest{"123": {
				EnvironmentID: "123",
				Metrics: clientgen.Metrics{
					TargetData:  &[]clientgen.TargetData{(*env123MetricsFlag1.TargetData)[0]},
					MetricsData: &[]clientgen.MetricsData{(*env123MetricsFlag1.MetricsData)[0], (*env123MetricsFlag2.MetricsData)[0]},
				},
			},
				"456": env456MetricsFlag1},
		},
		"Given metrics aren't enabled we don't save metrics sent to metricService": {
			metrics:  []domain.MetricsRequest{env123MetricsFlag1, env123MetricsFlag2, env456MetricsFlag1},
			enabled:  false,
			expected: map[string]domain.MetricsRequest{},
		},
	}

	for desc, tc := range testCases {
		tc := tc
		t.Run(desc, func(t *testing.T) {

			metricService := MetricService{metrics: map[string]domain.MetricsRequest{}, enabled: tc.enabled}

			for _, metric := range tc.metrics {
				metricService.StoreMetrics(context.Background(), metric)
			}

			actual := metricService.metrics

			assert.Equal(t, tc.expected, actual)

		})
	}
}

func TestMetricService_SendMetrics(t *testing.T) {
	postMetricsCount := 0
	testCases := map[string]struct {
		metrics              map[string]domain.MetricsRequest
		tokens               map[string]string
		expectedMetricsCount int
		postMetricsWithResp  func(environment string) (*clientgen.PostMetricsResponse, error)
	}{
		"Given I send one environments metrics successfully": {
			metrics:              map[string]domain.MetricsRequest{"123": env123MetricsFlag1},
			tokens:               map[string]string{"123": defaultToken},
			expectedMetricsCount: 1,
			postMetricsWithResp: func(environment string) (*clientgen.PostMetricsResponse, error) {
				postMetricsCount++
				return &clientgen.PostMetricsResponse{
					HTTPResponse: &http.Response{StatusCode: 200},
				}, nil
			},
		},
		"Given I have an error sending metrics for one env": {
			metrics:              map[string]domain.MetricsRequest{"123": env123MetricsFlag1},
			tokens:               map[string]string{"123": defaultToken},
			expectedMetricsCount: 1,
			postMetricsWithResp: func(environment string) (*clientgen.PostMetricsResponse, error) {
				postMetricsCount++
				return nil, fmt.Errorf("stuff went wrong")
			},
		},
		"Given I have 2 environments and the first errors we still send metrics for second env": {
			metrics:              map[string]domain.MetricsRequest{"123": env123MetricsFlag1, "456": env456MetricsFlag1},
			tokens:               map[string]string{"123": defaultToken, "456": defaultToken},
			expectedMetricsCount: 2,
			postMetricsWithResp: func(environment string) (*clientgen.PostMetricsResponse, error) {
				postMetricsCount++
				if environment == "123" {
					return nil, fmt.Errorf("stuff went wrong")
				}
				return &clientgen.PostMetricsResponse{HTTPResponse: &http.Response{StatusCode: 200}}, nil
			},
		},
		"Given I have 2 environments and missing a token for the first we skip it": {
			metrics:              map[string]domain.MetricsRequest{"123": env123MetricsFlag1, "456": env456MetricsFlag1},
			tokens:               map[string]string{"456": defaultToken},
			expectedMetricsCount: 1,
			postMetricsWithResp: func(environment string) (*clientgen.PostMetricsResponse, error) {
				postMetricsCount++
				return &clientgen.PostMetricsResponse{HTTPResponse: &http.Response{StatusCode: 200}}, nil
			},
		},
		"Given I have 2 environments and the first returns non 200 we still send metrics for second env": {
			metrics:              map[string]domain.MetricsRequest{"123": env123MetricsFlag1, "456": env456MetricsFlag1},
			tokens:               map[string]string{"123": defaultToken, "456": defaultToken},
			expectedMetricsCount: 2,
			postMetricsWithResp: func(environment string) (*clientgen.PostMetricsResponse, error) {
				postMetricsCount++
				if environment == "123" {
					return &clientgen.PostMetricsResponse{HTTPResponse: &http.Response{StatusCode: 500}}, nil
				}
				return &clientgen.PostMetricsResponse{HTTPResponse: &http.Response{StatusCode: 200}}, nil
			},
		},
	}

	for desc, tc := range testCases {
		t.Run(desc, func(t *testing.T) {
			postMetricsCount = 0
			logger, _ := log.NewStructuredLogger(true)
			metricsService, _ := NewMetricService(logger, defaultMetricsURL, defaultAccount, tc.tokens, true)
			metricsService.metrics = tc.metrics
			metricsService.client = mockService{postMetricsWithResp: tc.postMetricsWithResp}

			metricsService.SendMetrics(context.Background(), "1")

			// check metrics are cleared after sending
			actual := metricsService.metrics
			assert.Equal(t, map[string]domain.MetricsRequest{}, actual)
			// check how many times post metrics were called
			assert.Equal(t, tc.expectedMetricsCount, postMetricsCount)
		})
	}
}

func TestMetricService_addAuthToken(t *testing.T) {
	testCases := map[string]struct {
		token              string
		expectedAuthHeader string
		expectedErr        error
	}{
		"Given valid token exists in context then Authorization header is added to request": {
			token:              defaultToken,
			expectedAuthHeader: fmt.Sprintf("Bearer %s", defaultToken),
			expectedErr:        nil,
		},
		"Given no token in context error is returned": {
			token:              "",
			expectedAuthHeader: "",
			expectedErr:        fmt.Errorf("no auth token exists in context"),
		},
	}

	for desc, tc := range testCases {
		t.Run(desc, func(t *testing.T) {
			// create empty request
			req, _ := http.NewRequest("GET", "url", nil)

			// create context and add token to it
			ctx := context.Background()
			ctx = context.WithValue(ctx, tokenKey, tc.token)

			// check metrics are cleared after sending
			err := addAuthToken(ctx, req)

			// get auth header from updated request
			assert.Equal(t, tc.expectedAuthHeader, req.Header.Get("Authorization"))
			// check how many times post metrics were called
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}
