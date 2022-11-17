/*
Copyright 2022 The Kubernetes Authors All rights reserved.

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

package store

import (
	"context"
	"strings"

	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	basemetrics "k8s.io/component-base/metrics"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func customResourceDefinitionMetricFamilies() []generator.FamilyGenerator {
	getCRDVersions := func(crds []v1.CustomResourceDefinitionVersion) string {
		var versions []string
		for _, crd := range crds {
			versions = append(versions, crd.String())
		}
		return strings.Join(versions, ",")
	}
	return []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			"kube_customresourcedefinition_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapCustomResourceDefinitionFunc(func(crd *v1.CustomResourceDefinition) *metric.Family {
				var ms []*metric.Metric

				if !crd.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(crd.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_customresourcedefinition_info",
			"Information about customresourcedefinition.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapCustomResourceDefinitionFunc(func(crd *v1.CustomResourceDefinition) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"group", "versions", "kinds", "scope"},
							LabelValues: []string{crd.Spec.Group, getCRDVersions(crd.Spec.Versions), crd.Spec.Names.Kind, string(crd.Spec.Scope)},
							Value:       1,
						},
					},
				}
			}),
		),
	}
}

func wrapCustomResourceDefinitionFunc(f func(*v1.CustomResourceDefinition) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		crd := obj.(*v1.CustomResourceDefinition)

		metricFamily := f(crd)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descDeploymentLabelsDefaultLabels, []string{crd.Namespace, crd.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createCustomResourceDefinitionListWatch(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.ApiextensionsV1().CustomResourceDefinitions().List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.ApiextensionsV1().CustomResourceDefinitions().Watch(context.TODO(), opts)
		},
	}
}
