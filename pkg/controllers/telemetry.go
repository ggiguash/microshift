/*
Copyright © 2025 MicroShift Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/openshift/microshift/pkg/config"
	"github.com/openshift/microshift/pkg/telemetry"
	"github.com/openshift/microshift/pkg/util"
	"k8s.io/klog/v2"
)

const (
	// This file path must match the one in packaging/crio.conf.d/10-microshift_*.conf
	// entry for global_auth_file.
	PullSecretFilePath = "/etc/crio/openshift-pull-secret" // #nosec G101
)

type TelemetryManager struct {
	config *config.Config
}

func NewTelemetryManager(cfg *config.Config) *TelemetryManager {
	return &TelemetryManager{
		config: cfg,
	}
}

func (t *TelemetryManager) Name() string { return "telemetry-manager" }
func (t *TelemetryManager) Dependencies() []string {
	return []string{"kube-apiserver", "cluster-id-manager"}
}

func (t *TelemetryManager) Run(ctx context.Context, ready chan<- struct{}, stopped chan<- struct{}) error {
	// The service manager expects a service to get ready eventually before a timeout or else
	// MicroShift fails to start correctly. Stopping a service is relevant only when stopping
	// MicroShift, therefore if Telemetry is disabled we need to signal a fake readiness (even
	// though it is technically correct, the service is ready even if it is not going to do anything).
	// For stoppin the service we need to close the stopped channel or else MicroShift fails
	// to stop gracefully.
	defer close(stopped)
	if t.config.Telemetry.Status == config.StatusDisabled {
		klog.Info("Telemetry is disabled")
		close(ready)
		return nil
	}
	connected, err := util.HasDefaultRoute()
	if err != nil {
		close(ready)
		return fmt.Errorf("unable to check default routes: %v", err)
	}
	if !connected {
		klog.Info("Disconnected cluster detected, telemetry disabled")
		close(ready)
		return nil
	}
	clusterId, err := GetClusterId()
	if err != nil {
		close(ready)
		return fmt.Errorf("unable to get cluster id: %v", err)
	}
	close(ready)

	client := telemetry.NewTelemetryClient(t.config.Telemetry.Endpoint, clusterId, t.config.Telemetry.Proxy)

	collectAndSend := func() {
		reqCtx, reqCancel := context.WithTimeout(ctx, 10*time.Second)
		defer reqCancel()
		pullSecret, err := readPullSecret()
		if err != nil {
			klog.Errorf("Unable to get pull secret: %v", err)
			return
		}
		metrics, err := client.Collect(reqCtx, t.config)
		if err != nil {
			klog.Errorf("Failed to collect metrics: %v", err)
			return
		}
		klog.Infof("Collected telemetry data: %+v", metrics)
		if err := client.Send(reqCtx, pullSecret, metrics); err != nil {
			klog.Errorf("Failed to send metrics: %v", err)
		}
	}

	klog.Infof("MicroShift telemetry starting, sending first metrics collection. Cluster Id: %v", clusterId)
	collectAndSend()

	ticker := time.NewTicker(time.Hour)
	for {
		select {
		case <-ctx.Done():
			klog.Infof("MicroShift stopping, metrics shutting down")
			return nil
		case <-ticker.C:
			klog.Infof("Collect telemetry data to report back")
			collectAndSend()
		}
	}
}

func readPullSecret() (string, error) {
	data, err := os.ReadFile(PullSecretFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}
	var jsonData map[string]interface{}
	err = json.Unmarshal(data, &jsonData)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON: %v", err)
	}
	auths, ok := jsonData["auths"]
	if !ok {
		return "", fmt.Errorf("auths not found")
	}
	cloudOpenshiftCom, ok := auths.(map[string]interface{})["cloud.openshift.com"]
	if !ok {
		return "", fmt.Errorf("cloud.openshift.com not found")
	}
	auth, ok := cloudOpenshiftCom.(map[string]interface{})["auth"]
	if !ok {
		return "", fmt.Errorf("auth not found")
	}
	return auth.(string), nil
}
