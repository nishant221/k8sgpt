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
	"context"
	"fmt"
	"log"

	//"github.com/aws/aws-sdk-go/aws/client"
	"github.com/k8sgpt-ai/k8sgpt/pkg/common"
	"github.com/k8sgpt-ai/k8sgpt/pkg/util"

	// rollout "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	// 	"github.com/k8sgpt-ai/k8sgpt/pkg/util"
	// 	appsv1 "k8s.io/api/apps/v1"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rollout "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RolloutAnalyzer struct{}

func (RolloutAnalyzer) Analyze(a common.Analyzer) ([]common.Result, error) {

	kind := "Rollout"

	AnalyzerErrorsMetric.DeletePartialMatch(map[string]string{
		"analyzer_name": kind,
	})

	var kclient = GetClient()

	// roll := &rollout.Rollout{}
	// err := kclient.Get(context.TODO(), client.ObjectKey{
	// 	Namespace: "hackathon-demo",
	// 	Name:      "rollouts-demo",
	// }, roll)

	// fmt.Println(roll)

	rolloutlist := &rollout.RolloutList{}
	err := kclient.List(context.TODO(), rolloutlist)

	if err != nil {
		return nil, err
	}

	var preAnalysis = map[string]common.PreAnalysis{}

	for _, rollout := range rolloutlist.Items {
		var failures []common.Failure
		if *rollout.Spec.Replicas != rollout.Status.ReadyReplicas {
			failures = append(failures, common.Failure{
				Text: fmt.Sprintf("Rollout  %s/%s has %d replicas but %d are available", rollout.Namespace, rollout.Name, *rollout.Spec.Replicas, rollout.Status.ReadyReplicas),
				Sensitive: []common.Sensitive{
					{
						Unmasked: rollout.Namespace,
						Masked:   util.MaskString(rollout.Namespace),
					},
					{
						Unmasked: rollout.Name,
						Masked:   util.MaskString(rollout.Name),
					},
				}})
		}
		if len(failures) > 0 {
			preAnalysis[fmt.Sprintf("%s/%s", rollout.Namespace, rollout.Name)] = common.PreAnalysis{
				FailureDetails: failures,
				Rollout:        rollout,
			}
			AnalyzerErrorsMetric.WithLabelValues(kind, rollout.Name, rollout.Namespace).Set(float64(len(failures)))
		}

	}

	for key, value := range preAnalysis {
		var currentAnalysis = common.Result{
			Kind:  kind,
			Name:  key,
			Error: value.FailureDetails,
		}

		a.Results = append(a.Results, currentAnalysis)

	}

	return a.Results, nil
}

func GetClient() client.Client {
	scheme := runtime.NewScheme()
	rollout.AddToScheme(scheme)
	kubeconfig := ctrl.GetConfigOrDie()
	controllerClient, err := client.New(kubeconfig, client.Options{Scheme: scheme})
	if err != nil {
		log.Fatal(err)
		return nil
	}
	return controllerClient
}
