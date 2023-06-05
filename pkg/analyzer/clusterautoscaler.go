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

	"github.com/k8sgpt-ai/k8sgpt/pkg/common"
	"github.com/k8sgpt-ai/k8sgpt/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// 	"github.com/k8sgpt-ai/k8sgpt/pkg/util"
	// 	appsv1 "k8s.io/api/apps/v1"
	// 	corev1 "k8s.io/api/core/v1"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterAutoscalerAnalyzer struct{}

func (ClusterAutoscalerAnalyzer) Analyze(a common.Analyzer) ([]common.Result, error) {

	kind := "Pod"

	AnalyzerErrorsMetric.DeletePartialMatch(map[string]string{
		"analyzer_name": kind,
	})

	// deployment, err := a.Client.GetClient().AppsV1().Deployments(a.Namespace).Get(a.Context, name, metav1.GetOptions{})
	// list, err := a.Client.GetClient().CoreV1().Pods(a.Namespace).List(a.Context, metav1.ListOptions{})

	podlist, err := a.Client.GetClient().CoreV1().Pods(a.Namespace).List(a.Context, metav1.ListOptions{})

	//fmt.Println(podlist)

	if err != nil {
		return nil, err
	}

	// labelSelector := fmt.Sprintf("namespace=%s, param2=%s", version, param2)

	var preAnalysis = map[string]common.PreAnalysis{}

	for _, pod := range podlist.Items {
		var failures []common.Failure
		if pod.Status.Phase == "Pending" {

			//fmt.Println("xxxxxxx")
			//fmt.Println(pod.Name)
			// parse the event log and append details
			//evt, err := FetchLatestEvent(a.Context, a.Client, pod.Namespace, pod.Name)

			// get the list of events

			events, err := a.Client.GetClient().CoreV1().Events(a.Namespace).List(a.Context,
				metav1.ListOptions{
					FieldSelector: "involvedObject.name=" + pod.Name,
				})

			//fmt.Println(events)

			if err != nil {
				return nil, err
			}

			for _, evt := range events.Items {

				if evt.Reason == "NotTriggerScaleUp" && evt.Message != "" {
					failures = append(failures, common.Failure{
						Text:      evt.Message,
						Sensitive: []common.Sensitive{},
					})
				}
			}
		}
		if len(failures) > 0 {
			preAnalysis[fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)] = common.PreAnalysis{
				Pod:            pod,
				FailureDetails: failures,
			}
			AnalyzerErrorsMetric.WithLabelValues(kind, pod.Name, pod.Namespace).Set(float64(len(failures)))
		}
	}

	for key, value := range preAnalysis {
		var currentAnalysis = common.Result{
			Kind:  kind,
			Name:  key,
			Error: value.FailureDetails,
		}

		parent, _ := util.GetParent(a.Client, value.Pod.ObjectMeta)
		currentAnalysis.ParentObject = parent
		a.Results = append(a.Results, currentAnalysis)
	}
	return a.Results, nil
}
