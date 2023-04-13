// IN_TREE_POC_IMPL //

package store

import (
	"os"
	"os/exec"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"

	"github.com/rexagod/ksm-rpc-plugin-poc/shared"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func deploymentCollectorFromPlugin() (shared.MetricFamilies, *plugin.Client) {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		Output:     os.Stderr,
		JSONFormat: true,
	})

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "KSM_PLUGIN",
			MagicCookieValue: "deployment_collector",
		},
		Plugins: map[string]plugin.Plugin{
			"deployment_collector": &shared.MetricFamiliesPlugin{},
		},
		Cmd:    exec.Command("/home/rexagod/ksm-rpc-plugin-poc/plugin/ksm-deployment-collector"),
		Logger: logger,
	})

	rpcClient, err := client.Client()
	if err != nil {
		panic(err)
	}

	raw, err := rpcClient.Dispense("deployment_collector")
	if err != nil {
		panic(err)
	}

	return raw.(shared.MetricFamilies), client
}

func deploymentCollectorFromPluginWrapper(a, b []string) []generator.FamilyGenerator {
	dc, c := deploymentCollectorFromPlugin()
	defer c.Kill()
	resolvedFamilies := dc.XMetricFamilies(a, b)
	return resolvedFamilies
}
