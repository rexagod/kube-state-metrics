/*
Copyright 2025 The Kubernetes Authors All rights reserved.

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

package e2e

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	"k8s.io/kube-state-metrics/v2/internal"
	"k8s.io/kube-state-metrics/v2/internal/discovery"
	"k8s.io/kube-state-metrics/v2/pkg/options"
	ksmFramework "k8s.io/kube-state-metrics/v2/tests/e2e/framework"
)

func TestCRSFactoryDeduplication(t *testing.T) {
	const (
		assetDir        = "testdata/crs_deduplication"
		populateTimeout = 30 * time.Second
	)

	m := &struct {
		crsConfig string
		vpaCRD    string
		vpaCR     string
		vpaDeploy string
	}{
		crsConfig: assetDir + "/customresourcestate-config.yaml",
		vpaCRD:    assetDir + "/verticalpodautoscaler-crd.yaml",
		vpaCR:     assetDir + "/verticalpodautoscaler.yaml",
		vpaDeploy: assetDir + "/verticalpodautoscaler-target-deployment.yaml",
	}

	defer func() {
		klog.InfoS("cleaning up test resources")
		_ = exec.Command("kubectl", "delete", "-f", m.vpaCR, "--ignore-not-found").Run()     //nolint:gosec
		_ = exec.Command("kubectl", "delete", "-f", m.vpaDeploy, "--ignore-not-found").Run() //nolint:gosec
		_ = exec.Command("kubectl", "delete", "-f", m.vpaCRD, "--ignore-not-found").Run()    //nolint:gosec
	}()

	opts := options.NewOptions()
	opts.AddFlags(options.InitCommand)
	opts.CustomResourceConfigFile = m.crsConfig
	opts.CustomResourcesOnly = true
	kubeconfig, found := os.LookupEnv("KUBECONFIG")
	if !found {
		t.Fatalf("KUBECONFIG environment variable not set")
	}
	opts.Kubeconfig = kubeconfig
	if err := opts.Parse(); err != nil {
		t.Fatalf("failed to parse options: %v", err)
	}

	go internal.RunKubeStateMetricsWrapper(opts)

	err := wait.PollUntilContextTimeout(context.TODO(), 1*time.Second, 20*time.Second, true, func(_ context.Context) (bool, error) {
		conn, err := net.Dial("tcp", "localhost:8080")
		if err != nil {
			return false, nil
		}
		return conn.Close() == nil, nil
	})
	if err != nil {
		t.Fatalf("failed while waiting for port 8080 to come up: %v", err)
	}

	f, err := ksmFramework.New("http://localhost:8080", "http://localhost:8081")
	if err != nil {
		t.Fatalf("failed to create test framework: %v", err)
	}

	if err = exec.Command("kubectl", "apply", "-f", m.vpaCRD).Run(); err != nil { //nolint:gosec
		t.Fatalf("failed to apply VPA CRD: %v", err)
	}
	if err = exec.Command("kubectl", "apply", "-f", m.vpaDeploy).Run(); err != nil { //nolint:gosec
		t.Fatalf("failed to apply VPA target deployment: %v", err)
	}
	if err = exec.Command("kubectl", "apply", "-f", m.vpaCR).Run(); err != nil { //nolint:gosec
		t.Fatalf("failed to apply VPA CR: %v", err)
	}

	testMetric := "kube_customresource_verticalpodautoscaler_spec_updatepolicy_updatemode"
	testVPAName := "hamster-vpa"

	err = wait.PollUntilContextTimeout(context.TODO(), discovery.Interval, populateTimeout, true, func(_ context.Context) (bool, error) {
		var buf bytes.Buffer
		if err := f.KsmClient.Metrics(&buf); err != nil {
			return false, nil
		}
		return strings.Contains(buf.String(), testMetric), nil
	})
	if err != nil {
		t.Fatalf("VPA metrics not available: %v", err)
	}

	for range 5 {
		t.Log("!!!DELETING CRD!!! (list/watch errors are expected here)")
		if err := exec.Command("kubectl", "delete", "-f", m.vpaCRD).Run(); err != nil { //nolint:gosec
			t.Logf("CRD deletion failed: %v", err)
		}
		time.Sleep(discovery.Interval)

		t.Log("!!!RECREATING CRD!!!")
		if err := exec.Command("kubectl", "apply", "-f", m.vpaCRD).Run(); err != nil { //nolint:gosec
			t.Fatalf("CRD recreation failed: %v", err)
		}
		t.Log("!!!RECREATING VPA CR!!!")
		if err := exec.Command("kubectl", "apply", "-f", m.vpaCR).Run(); err != nil { //nolint:gosec
			t.Fatalf("VPA CR recreation failed: %v", err)
		}
		time.Sleep(discovery.Interval + time.Second)
	}

	time.Sleep(discovery.Interval * 3)

	var buf bytes.Buffer
	if err := f.KsmClient.Metrics(&buf); err != nil {
		t.Fatalf("failed to fetch metrics: %v", err)
	}
	metricsOutput := buf.String()

	metricLinePattern := fmt.Sprintf(`%s{`, testMetric)
	vpaPattern := fmt.Sprintf(`verticalpodautoscaler="%s"`, testVPAName)

	var duplicateCount int
	for _, line := range strings.Split(metricsOutput, "\n") {
		if strings.Contains(line, metricLinePattern) && strings.Contains(line, vpaPattern) {
			duplicateCount++
		}
	}

	klog.InfoS("metric line counts for VPA", "vpa", testVPAName, "count", duplicateCount)

	if duplicateCount != 4 {
		t.Errorf("expected 4 metric line for VPA %s, got %d (duplicate stores detected)", testVPAName, duplicateCount)
	}

	t.Logf("metrics:\n%v", metricsOutput)
}
