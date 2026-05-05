package portfolio

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type promResult struct {
	Labels map[string]string
	Value  float64
}

type promClient struct {
	base   string
	client *http.Client
}

func newPromClient(baseURL string) *promClient {
	return &promClient{base: baseURL, client: &http.Client{Timeout: 10 * time.Second}}
}

func (p *promClient) queryVector(ctx context.Context, query string) ([]promResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.base+"/api/v1/query", nil)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = url.Values{"query": {query}}.Encode()

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("prometheus: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var env struct {
		Status string `json:"status"`
		Error  string `json:"error,omitempty"`
		Data   struct {
			Result []struct {
				Metric map[string]string  `json:"metric"`
				Value  [2]json.RawMessage `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("prometheus decode: %w", err)
	}
	if env.Status != "success" {
		return nil, fmt.Errorf("prometheus: %s", env.Error)
	}
	out := make([]promResult, 0, len(env.Data.Result))
	for _, r := range env.Data.Result {
		var s string
		_ = json.Unmarshal(r.Value[1], &s)
		v, _ := strconv.ParseFloat(s, 64)
		out = append(out, promResult{Labels: r.Metric, Value: v})
	}
	return out, nil
}

func (p *promClient) queryScalar(ctx context.Context, query string) (float64, bool, error) {
	rows, err := p.queryVector(ctx, query)
	if err != nil || len(rows) == 0 {
		return 0, false, err
	}
	return rows[0].Value, true, nil
}
