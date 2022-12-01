/*
Copyright 2018 The Kubernetes Authors All rights reserved.

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

package options

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/prometheus/common/version"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

// Options are the configurable parameters for kube-state-metrics.
type Options struct {
	AnnotationsAllowList     LabelsAllowList `yaml:"annotations_allow_list"`
	Apiserver                string          `yaml:"apiserver"`
	CustomResourceConfig     string          `yaml:"custom_resource_config"`
	CustomResourceConfigFile string          `yaml:"custom_resource_config_file"`
	CustomResourcesOnly      bool            `yaml:"custom_resources_only"`
	EnableGZIPEncoding       bool            `yaml:"enable_gzip_encoding"`
	Help                     bool            `yaml:"help"`
	Host                     string          `yaml:"host"`
	Kubeconfig               string          `yaml:"kubeconfig"`
	LabelsAllowList          LabelsAllowList `yaml:"labels_allow_list"`
	MetricAllowlist          MetricSet       `yaml:"metric_allowlist"`
	MetricDenylist           MetricSet       `yaml:"metric_denylist"`
	MetricOptInList          MetricSet       `yaml:"metric_opt_in_list"`
	Namespace                string          `yaml:"namespace"`
	Namespaces               NamespaceList   `yaml:"namespaces"`
	NamespacesDenylist       NamespaceList   `yaml:"namespaces_denylist"`
	Node                     NodeType        `yaml:"node"`
	Pod                      string          `yaml:"pod"`
	Port                     int             `yaml:"port"`
	Resources                ResourceSet     `yaml:"resources"`
	Shard                    int32           `yaml:"shard"`
	TLSConfig                string          `yaml:"tls_config"`
	TelemetryHost            string          `yaml:"telemetry_host"`
	TelemetryPort            int             `yaml:"telemetry_port"`
	TotalShards              int             `yaml:"total_shards"`
	UseAPIServerCache        bool            `yaml:"use_api_server_cache"`
	Version                  bool            `yaml:"version"`

	Config string

	cmd *cobra.Command
}

// GetOptsConfigFile is the getter for --options-config-file value.
func GetOptsConfigFile(opt Options) string {
	return opt.Config
}

// NewOptions returns a new instance of `Options`.
func NewOptions() *Options {
	return &Options{
		Resources:            ResourceSet{},
		MetricAllowlist:      MetricSet{},
		MetricDenylist:       MetricSet{},
		MetricOptInList:      MetricSet{},
		AnnotationsAllowList: LabelsAllowList{},
		LabelsAllowList:      LabelsAllowList{},
	}
}

// AddFlags populated the Options struct from the command line arguments passed.
func (o *Options) AddFlags(cmd *cobra.Command) {
	o.cmd = cmd

	completionCommand.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if shellPath, ok := os.LookupEnv("SHELL"); ok {
			shell := shellPath[strings.LastIndex(shellPath, "/")+1:]
			fmt.Println(FetchLoadInstructions(shell))
		} else {
			fmt.Println("SHELL environment variable not set, falling back to bash")
			fmt.Println(FetchLoadInstructions("bash"))
		}
		klog.FlushAndExit(klog.ExitFlushTimeout, 0)
	})

	versionCommand := &cobra.Command{
		Use:   "version",
		Short: "Print version information.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s\n", version.Print("kube-state-metrics"))
			klog.FlushAndExit(klog.ExitFlushTimeout, 0)
		},
	}

	cmd.AddCommand(completionCommand, versionCommand)

	o.cmd.Flags().Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		o.cmd.Flags().PrintDefaults()
	}

	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)
	o.cmd.Flags().AddGoFlagSet(klogFlags)
	_ = o.cmd.Flags().Lookup("logtostderr").Value.Set("true")
	o.cmd.Flags().Lookup("logtostderr").DefValue = "true"
	o.cmd.Flags().Lookup("logtostderr").NoOptDefVal = "true"

	autoshardingNotice := "When set, it is expected that --pod and --pod-namespace are both set. Most likely this should be passed via the downward API. This is used for auto-detecting sharding. If set, this has preference over statically configured sharding. This is experimental, it may be removed without notice."

	o.cmd.Flags().BoolVar(&o.CustomResourcesOnly, "custom-resource-state-only", false, "Only provide Custom Resource State metrics (experimental)")
	o.cmd.Flags().BoolVar(&o.EnableGZIPEncoding, "enable-gzip-encoding", false, "Gzip responses when requested by clients via 'Accept-Encoding: gzip' header.")
	o.cmd.Flags().BoolVarP(&o.Help, "help", "h", false, "Print Help text")
	o.cmd.Flags().BoolVarP(&o.UseAPIServerCache, "use-apiserver-cache", "", false, "Sets resourceVersion=0 for ListWatch requests, using cached resources from the apiserver instead of an etcd quorum read.")
	o.cmd.Flags().BoolVarP(&o.Version, "version", "", false, "kube-state-metrics build version information")
	o.cmd.Flags().Int32Var(&o.Shard, "shard", int32(0), "The instances shard nominal (zero indexed) within the total number of shards. (default 0)")
	o.cmd.Flags().IntVar(&o.Port, "port", 8080, `Port to expose metrics on.`)
	o.cmd.Flags().IntVar(&o.TelemetryPort, "telemetry-port", 8081, `Port to expose kube-state-metrics self metrics on.`)
	o.cmd.Flags().IntVar(&o.TotalShards, "total-shards", 1, "The total number of shards. Sharding is disabled when total shards is set to 1.")
	o.cmd.Flags().StringVar(&o.Apiserver, "apiserver", "", `The URL of the apiserver to use as a master`)
	o.cmd.Flags().StringVar(&o.CustomResourceConfig, "custom-resource-state-config", "", "Inline Custom Resource State Metrics config YAML (experimental)")
	o.cmd.Flags().StringVar(&o.CustomResourceConfigFile, "custom-resource-state-config-file", "", "Path to a Custom Resource State Metrics config file (experimental)")
	o.cmd.Flags().StringVar(&o.Host, "host", "::", `Host to expose metrics on.`)
	o.cmd.Flags().StringVar(&o.Kubeconfig, "kubeconfig", "", "Absolute path to the kubeconfig file")
	o.cmd.Flags().StringVar(&o.Namespace, "pod-namespace", "", "Name of the namespace of the pod specified by --pod. "+autoshardingNotice)
	o.cmd.Flags().StringVar(&o.Pod, "pod", "", "Name of the pod that contains the kube-state-metrics container. "+autoshardingNotice)
	o.cmd.Flags().StringVar(&o.TLSConfig, "tls-config", "", "Path to the TLS configuration file")
	o.cmd.Flags().StringVar(&o.TelemetryHost, "telemetry-host", "::", `Host to expose kube-state-metrics self metrics on.`)
	o.cmd.Flags().StringVar(&o.Config, "config", "", "Path to the kube-state-metrics options config file")
	o.cmd.Flags().StringVar((*string)(&o.Node), "node", "", "Name of the node that contains the kube-state-metrics pod. Most likely it should be passed via the downward API. This is used for daemonset sharding. Only available for resources (pod metrics) that support spec.nodeName fieldSelector. This is experimental.")
	o.cmd.Flags().Var(&o.AnnotationsAllowList, "metric-annotations-allowlist", "Comma-separated list of Kubernetes annotations keys that will be used in the resource' labels metric. By default the metric contains only name and namespace labels. To include additional annotations provide a list of resource names in their plural form and Kubernetes annotation keys you would like to allow for them (Example: '=namespaces=[kubernetes.io/team,...],pods=[kubernetes.io/team],...)'. A single '*' can be provided per resource instead to allow any annotations, but that has severe performance implications (Example: '=pods=[*]').")
	o.cmd.Flags().Var(&o.LabelsAllowList, "metric-labels-allowlist", "Comma-separated list of additional Kubernetes label keys that will be used in the resource' labels metric. By default the metric contains only name and namespace labels. To include additional labels provide a list of resource names in their plural form and Kubernetes label keys you would like to allow for them (Example: '=namespaces=[k8s-label-1,k8s-label-n,...],pods=[app],...)'. A single '*' can be provided per resource instead to allow any labels, but that has severe performance implications (Example: '=pods=[*]'). Additionally, an asterisk (*) can be provided as a key, which will resolve to all resources, i.e., assuming '--resources=deployments,pods', '=*=[*]' will resolve to '=deployments=[*],pods=[*]'.")
	o.cmd.Flags().Var(&o.MetricAllowlist, "metric-allowlist", "Comma-separated list of metrics to be exposed. This list comprises of exact metric names and/or regex patterns. The allowlist and denylist are mutually exclusive.")
	o.cmd.Flags().Var(&o.MetricDenylist, "metric-denylist", "Comma-separated list of metrics not to be enabled. This list comprises of exact metric names and/or regex patterns. The allowlist and denylist are mutually exclusive.")
	o.cmd.Flags().Var(&o.MetricOptInList, "metric-opt-in-list", "Comma-separated list of metrics which are opt-in and not enabled by default. This is in addition to the metric allow- and denylists")
	o.cmd.Flags().Var(&o.Namespaces, "namespaces", fmt.Sprintf("Comma-separated list of namespaces to be enabled. Defaults to %q", &DefaultNamespaces))
	o.cmd.Flags().Var(&o.NamespacesDenylist, "namespaces-denylist", "Comma-separated list of namespaces not to be enabled. If namespaces and namespaces-denylist are both set, only namespaces that are excluded in namespaces-denylist will be used.")
	o.cmd.Flags().Var(&o.Resources, "resources", fmt.Sprintf("Comma-separated list of Resources to be enabled. Defaults to %q", &DefaultResources))
}

// Parse parses the flag definitions from the argument list.
func (o *Options) Parse() error {
	err := o.cmd.Execute()
	return err
}

// Usage is the function called when an error occurs while parsing flags.
func (o *Options) Usage() {
	_ = o.cmd.Flags().FlagUsages()
}

// Validate validates arguments
func (o *Options) Validate() error {
	shardableResource := "pods"
	if o.Node == "" {
		return nil
	}
	for _, x := range o.Resources.AsSlice() {
		if x != shardableResource {
			return fmt.Errorf("resource %s can't be sharded by field selector spec.nodeName", x)
		}
	}
	return nil
}
