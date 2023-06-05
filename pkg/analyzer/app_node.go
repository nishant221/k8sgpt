/*
Copyright 2023 The K8sGPT Authors.
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

package analyzer

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/k8sgpt-ai/k8sgpt/pkg/common"
	"github.com/k8sgpt-ai/k8sgpt/pkg/util"
	"golang.org/x/exp/slices"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AppNodeAnalyzer struct{}

func (AppNodeAnalyzer) Analyze(a common.Analyzer) ([]common.Result, error) {

	kind := "Rollout"

	AnalyzerErrorsMetric.DeletePartialMatch(map[string]string{
		"analyzer_name": kind,
	})

	list, err := a.Client.GetClient().CoreV1().Pods(a.Namespace).List(a.Context, metav1.ListOptions{})
	listNode, errNode := a.Client.GetClient().CoreV1().Nodes().List(a.Context, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	if errNode != nil {
		return nil, err
	}

	var preAnalysis = map[string]common.PreAnalysis{}

	var podNodeList []string
	podNodeMap := make(map[string]string)

	for _, pod := range list.Items {
		podNodeList = append(podNodeList, pod.Spec.NodeName)
		podNodeMap[pod.Spec.NodeName] = podNodeMap[pod.Spec.NodeName] + pod.Name + " , "
	}

	for _, node := range listNode.Items {
		var failures []common.Failure
		if slices.Contains(podNodeList, node.Name) {
			for _, nodeCondition := range node.Status.Conditions {
				// https://kubernetes.io/docs/concepts/architecture/nodes/#condition
				switch nodeCondition.Type {
				case v1.NodeReady:
					if nodeCondition.Status == v1.ConditionTrue {
						break
					}
					failures = addPodNodeConditionFailure(failures, node.Name, nodeCondition)
				default:
					if nodeCondition.Status != v1.ConditionFalse {

						failures = addPodNodeConditionFailure(failures, node.Name, nodeCondition)
					}
				}
			}

		}
		if len(failures) > 0 {
			color.Magenta("%s pods can have problems as underlying node %s have failures", podNodeMap[node.Name], node.Name)
			preAnalysis[node.Name] = common.PreAnalysis{
				Node:           node,
				FailureDetails: failures,
			}
			AnalyzerErrorsMetric.WithLabelValues(kind, node.Name, "").Set(float64(len(failures)))
		}
	}

	for key, value := range preAnalysis {
		var currentAnalysis = common.Result{
			Kind:  kind,
			Name:  key,
			Error: value.FailureDetails,
		}

		parent, _ := util.GetParent(a.Client, value.Node.ObjectMeta)
		currentAnalysis.ParentObject = parent
		a.Results = append(a.Results, currentAnalysis)
	}

	return a.Results, nil
}

func addPodNodeConditionFailure(failures []common.Failure, nodeName string, nodeCondition v1.NodeCondition) []common.Failure {
	failures = append(failures, common.Failure{
		Text: fmt.Sprintf("%s has condition of type %s, reason %s: %s", nodeName, nodeCondition.Type, nodeCondition.Reason, nodeCondition.Message),
		Sensitive: []common.Sensitive{
			{
				Unmasked: nodeName,
				Masked:   util.MaskString(nodeName),
			},
		},
	})
	return failures
}
