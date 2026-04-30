package control

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	controlv1 "github.com/rayip/rayip/packages/proto/gen/go/rayip/control/v1"
	"github.com/rayip/rayip/services/node-agent/internal/runtime"
	"go.uber.org/zap"
)

type httpWelcome struct {
	NodeID          string `json:"node_id"`
	SessionID       string `json:"session_id"`
	LeaseTTLSeconds int    `json:"lease_ttl_seconds"`
	APIInstanceID   string `json:"api_instance_id"`
}

type httpPollResponse struct {
	RuntimeApply *runtime.Apply `json:"runtime_apply"`
}

func (c *Client) runHTTP(ctx context.Context) error {
	client := &http.Client{Timeout: 30 * time.Second}
	seq := uint64(1)
	runtimeInfo, err := c.discover()
	if err != nil {
		return err
	}
	probeInfo, _ := probePublicReachability(c.cfg.Probe, time.Now)
	if probeInfo == nil {
		probeInfo = fallbackProbe(c)
	}
	var welcome httpWelcome
	if err := c.postJSON(ctx, client, "/api/node-agent/hello", map[string]any{
		"node_code":         c.cfg.Node.Code,
		"enrollment_token":  c.cfg.Node.EnrollmentToken,
		"agent_version":     runtimeInfo.AgentVersion,
		"xray_version":      runtimeInfo.XrayVersion,
		"bundle_version":    runtimeInfo.BundleVersion,
		"capabilities":      runtimeInfo.Capabilities,
		"sequence":          seq,
		"lease_ttl_seconds": int64(c.cfg.Lease.TTL / time.Second),
		"runtime":           runtimeInfo,
		"probe":             probeInfo,
	}, &welcome); err != nil {
		return err
	}
	c.log.Info("registered with api",
		zapString("node_id", welcome.NodeID),
		zapString("session_id", welcome.SessionID),
		zapString("api_instance_id", welcome.APIInstanceID),
	)

	leaseTicker := time.NewTicker(c.cfg.Lease.Interval)
	defer leaseTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-leaseTicker.C:
			seq++
			runtimeInfo, err = c.discover()
			if err != nil {
				return err
			}
			probeInfo, _ = probePublicReachability(c.cfg.Probe, time.Now)
			if probeInfo == nil {
				probeInfo = fallbackProbe(c)
			}
			if err := c.postJSON(ctx, client, "/api/node-agent/lease", map[string]any{
				"node_id":           welcome.NodeID,
				"node_code":         c.cfg.Node.Code,
				"session_id":        welcome.SessionID,
				"agent_version":     runtimeInfo.AgentVersion,
				"xray_version":      runtimeInfo.XrayVersion,
				"bundle_version":    runtimeInfo.BundleVersion,
				"capabilities":      runtimeInfo.Capabilities,
				"sequence":          seq,
				"lease_ttl_seconds": int64(c.cfg.Lease.TTL / time.Second),
				"runtime":           runtimeInfo,
				"probe":             probeInfo,
			}, nil); err != nil {
				return err
			}
		default:
			apply, ok, err := c.pollHTTP(ctx, client, welcome.NodeID)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
			ack, applyErr := c.runtime.Apply(ctx, apply)
			if applyErr != nil {
				c.log.Warn("runtime apply failed", zapString("apply_id", apply.ApplyID), zapError(applyErr))
			}
			if err := c.postJSON(ctx, client, "/api/node-agent/runtime-apply-ack", ack, nil); err != nil {
				return err
			}
		}
	}
}

func fallbackProbe(c *Client) *controlv1.NodeProbeObservation {
	return &controlv1.NodeProbeObservation{
		ProbePort:          c.cfg.Probe.Port,
		ProbeProtocols:     append([]string(nil), c.cfg.Probe.Protocols...),
		CheckedAtUnixMilli: time.Now().UTC().UnixMilli(),
		ScanHost:           c.cfg.Probe.ScanHost,
	}
}

func zapString(key string, value string) zap.Field {
	return zap.String(key, value)
}

func zapError(err error) zap.Field {
	return zap.Error(err)
}

func (c *Client) pollHTTP(ctx context.Context, client *http.Client, nodeID string) (runtime.Apply, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.API.HTTPURL+"/api/node-agent/nodes/"+nodeID+"/poll", nil)
	if err != nil {
		return runtime.Apply{}, false, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return runtime.Apply{}, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return runtime.Apply{}, false, fmt.Errorf("poll returned HTTP %d", resp.StatusCode)
	}
	var payload httpPollResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return runtime.Apply{}, false, err
	}
	if payload.RuntimeApply == nil || payload.RuntimeApply.ApplyID == "" {
		return runtime.Apply{}, false, nil
	}
	return *payload.RuntimeApply, true, nil
}

func (c *Client) postJSON(ctx context.Context, client *http.Client, path string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.API.HTTPURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", uuid.NewString())
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s returned HTTP %d", path, resp.StatusCode)
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}
		return err
	}
	return nil
}
