package arouter

import (
	"context"
	"fmt"
	"net/http"
)

// --- Usage & Analytics API ---
//
// These methods query usage data for the authenticated tenant.
// Endpoint: /api/usage

// GetUsageSummary returns aggregated usage statistics for the given time range.
func (c *Client) GetUsageSummary(ctx context.Context, query *UsageQuery) (*UsageSummary, error) {
	httpReq, err := c.newRequest(ctx, http.MethodGet, "/api/usage/summary", nil)
	if err != nil {
		return nil, err
	}
	applyUsageQuery(httpReq, query)

	var resp UsageSummary
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetUsageTimeSeries returns usage data as a time series for the given time range.
func (c *Client) GetUsageTimeSeries(ctx context.Context, query *UsageQuery) (*UsageTimeSeries, error) {
	httpReq, err := c.newRequest(ctx, http.MethodGet, "/api/usage/timeseries", nil)
	if err != nil {
		return nil, err
	}
	applyUsageQuery(httpReq, query)

	var resp UsageTimeSeries
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func applyUsageQuery(req *http.Request, query *UsageQuery) {
	if query == nil {
		return
	}
	q := req.URL.Query()
	if query.StartTime != "" {
		q.Set("start_time", query.StartTime)
	}
	if query.EndTime != "" {
		q.Set("end_time", query.EndTime)
	}
	if query.ProviderID != "" {
		q.Set("provider_id", query.ProviderID)
	}
	if query.Model != "" {
		q.Set("model", query.Model)
	}
	if query.KeyID != "" {
		q.Set("key_id", query.KeyID)
	}
	if query.Granularity != "" {
		q.Set("granularity", query.Granularity)
	}
	req.URL.RawQuery = q.Encode()
}

// Convenience helper for formatting time range queries.
func UsageQueryLast30Days() *UsageQuery {
	return &UsageQuery{}
}

// UsageQueryWithRange creates a UsageQuery with explicit start/end times (RFC3339).
func UsageQueryWithRange(startTime, endTime string) *UsageQuery {
	return &UsageQuery{
		StartTime: startTime,
		EndTime:   endTime,
	}
}

// String returns a human-readable summary of the usage data.
func (s *UsageSummary) String() string {
	return fmt.Sprintf(
		"Requests: %d | Tokens: %d (in: %d, out: %d) | Cost: $%.4f",
		s.TotalRequests, s.TotalTokens, s.TotalInputTokens, s.TotalOutputTokens, s.EstimatedCostUSD,
	)
}
