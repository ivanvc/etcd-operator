/*
Copyright 2024.

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
	"context"
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	apiextensionsV1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// Sample Feature-based test with e2e-framework
func TestBasicFeature(t *testing.T) {
	feature := features.New("etcd-operator-controller")

	feature.Assess("Check if the crd exists",
		func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			t.Log("Assessing the state of the cluster...")

			client := cfg.Client()
			_ = apiextensionsV1.AddToScheme(client.Resources().GetScheme())

			var crd apiextensionsV1.CustomResourceDefinition
			if err := client.Resources().Get(ctx, "etcdclusters.operator.etcd.io", "", &crd); err != nil {
				t.Fatalf("Failed due to error: %s", err)
			}

			t.Log("Everything looks good!")
			return ctx
		})

	// 'testEnv' is the env.Environment you set up in TestMain
	_ = testEnv.Test(t, feature.Feature())
}
