package portfolio

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type k8sClient struct {
	base   string
	token  string
	client *http.Client
}

func newInClusterK8sClient() (*k8sClient, error) {
	tokenBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return nil, fmt.Errorf("read sa token: %w", err)
	}
	caBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err != nil {
		return nil, fmt.Errorf("read ca cert: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caBytes) {
		return nil, fmt.Errorf("failed to parse CA cert")
	}
	return &k8sClient{
		base:  "https://kubernetes.default.svc",
		token: strings.TrimSpace(string(tokenBytes)),
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{RootCAs: pool},
			},
		},
	}, nil
}

// Minimal response types — only fields the engine uses.

type NodeList struct {
	Items []struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
		Status struct {
			Conditions []struct {
				Type   string `json:"type"`
				Status string `json:"status"`
			} `json:"conditions"`
		} `json:"status"`
	} `json:"items"`
}

type PodList struct {
	Items []struct {
		Metadata struct {
			Name      string            `json:"name"`
			Namespace string            `json:"namespace"`
			Labels    map[string]string `json:"labels"`
		} `json:"metadata"`
		Status struct {
			Phase             string `json:"phase"`
			ContainerStatuses []struct {
				RestartCount int  `json:"restartCount"`
				Ready        bool `json:"ready"`
				LastState    struct {
					Terminated *struct {
						Reason string `json:"reason"`
					} `json:"terminated"`
				} `json:"lastState"`
			} `json:"containerStatuses"`
		} `json:"status"`
	} `json:"items"`
}

func (c *k8sClient) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("k8s: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("k8s: HTTP %d: %s", resp.StatusCode, body[:min(len(body), 200)])
	}
	return json.Unmarshal(body, out)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (c *k8sClient) ListNodes(ctx context.Context) (*NodeList, error) {
	var out NodeList
	return &out, c.get(ctx, "/api/v1/nodes", &out)
}

func (c *k8sClient) ListPods(ctx context.Context) (*PodList, error) {
	var out PodList
	return &out, c.get(ctx, "/api/v1/pods", &out)
}
