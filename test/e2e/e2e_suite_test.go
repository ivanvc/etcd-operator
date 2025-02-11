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
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/pkg/utils"
	"sigs.k8s.io/e2e-framework/support/kind"

	test_utils "go.etcd.io/etcd-operator/test/utils"
)

var (
	testEnv     env.Environment
	dockerImage = "etcd-operator-controller:current"
	namespace   = "etcd-operator-system"
)

func TestMain(m *testing.M) {
	testEnv = env.New()
	kindClusterName := "etcd-cluster"
	kindCluster := kind.NewCluster(kindClusterName)

	log.Println("Creating KinD cluster...")
	origWd, _ := os.Getwd()
	testEnv.Setup(
		// create namespace and deploy the etcd-operator
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			var err error

			// create KinD cluster
			ctx, err = envfuncs.CreateCluster(kindCluster, kindClusterName)(ctx, cfg)
			if err != nil {
				log.Printf("failed to create cluster: %s", err)
				return ctx, err
			}

			// create namespace
			ctx, err = envfuncs.CreateNamespace(namespace)(ctx, cfg)
			if err != nil {
				log.Printf("failed to create namespace: %s", err)
				return ctx, err
			}

			return ctx, nil
		},

		// prepare the resources
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			// change dir for Make file
			if err := os.Chdir("../../"); err != nil {
				log.Printf("Unable to set working directory: %s", err)
				return ctx, err
			}

			log.Println("Installing bin tools...")
			if p := utils.RunCommand(
				`make kustomize`,
			); p.Err() != nil {
				log.Printf("Failed to install kustomize binary: %s: %s", p.Err(), p.Out())
				return ctx, p.Err()
			}

			if p := utils.RunCommand(
				`make controller-gen`,
			); p.Err() != nil {
				log.Printf("Failed to install controller-gen binary: %s: %s", p.Err(), p.Out())
				return ctx, p.Err()
			}

			// gen manifest files
			log.Println("Generate manifests...")
			if p := utils.RunCommand(
				`make manifests`,
			); p.Err() != nil {
				log.Printf("Failed to generate manifests: %s: %s", p.Err(), p.Out())
				return ctx, p.Err()
			}

			// install crd
			log.Println("Install crd...")
			if p := utils.RunCommand(
				`make install`,
			); p.Err() != nil {
				log.Printf("Failed to generate manifests: %s: %s", p.Err(), p.Out())
				return ctx, p.Err()
			}

			// Build docker image
			log.Println("Building docker image...")
			if p := utils.RunCommand(fmt.Sprintf("make docker-build IMG=%s", dockerImage)); p.Err() != nil {
				log.Printf("Failed to build docker image: %s: %s", p.Err(), p.Out())
				return ctx, p.Err()
			}

			// Load docker image into kind
			log.Println("Loading docker image into kind cluster...")
			if err := kindCluster.LoadImage(ctx, dockerImage); err != nil {
				log.Printf("Failed to load image into kind: %s", err)
				return ctx, err
			}

			// set working directory test/e2e
			if err := os.Chdir(origWd); err != nil {
				log.Printf("Unable to set working directory: %s", err)
				return ctx, err
			}

			return ctx, nil
		},

		// install prometheus and cert-manager
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			log.Println("Installing prometheus operator...")
			if err := test_utils.InstallPrometheusOperator(); err != nil {
				log.Printf("Unable to install Prometheus operator: %s", err)
			}

			log.Println("Installing cert-manager...")
			if err := test_utils.InstallCertManager(); err != nil {
				log.Printf("Unable to install Cert Manager: %s", err)
			}

			// set working directory test/e2e
			if err := os.Chdir(origWd); err != nil {
				log.Printf("Unable to set working directory: %s", err)
				return ctx, err
			}

			return ctx, nil
		},

		// set up environment
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			// make sure you are on the orignal wd
			if err := os.Chdir(origWd); err != nil {
				log.Printf("Unable to set working directory: %s", err)
				return ctx, err
			}

			// change working directory for Make file
			if err := os.Chdir("../../"); err != nil {
				log.Printf("Unable to set working directory: %s", err)
				return ctx, err
			}

			// Deploy components
			log.Println("Deploying components...")
			log.Println("Deploying controller-manager resources...")
			if p := utils.RunCommand(
				`make deploy`,
			); p.Err() != nil {
				log.Printf("Failed to deploy resource configurations: %s: %s", p.Err(), p.Out())
				return ctx, p.Err()
			}

			// wait for controller to get ready
			log.Println("Waiting for controller-manager deployment to be available...")
			client := cfg.Client()
			if err := wait.For(
				conditions.New(client.Resources()).DeploymentAvailable("etcd-operator-controller-manager", "etcd-operator-system"),
				wait.WithTimeout(3*time.Minute),
				wait.WithInterval(10*time.Second),
			); err != nil {
				log.Printf("Timed out while waiting for etcd-operator-controller-manager deployment: %s", err)
				return ctx, err
			}
			// set working directory test/e2e
			if err := os.Chdir(origWd); err != nil {
				log.Printf("Unable to set working directory: %s", err)
				return ctx, err
			}

			return ctx, nil
		},
	)

	// Use the Environment.Finish method to define clean up steps
	testEnv.Finish(
		// cleanup environment
		func(ctx context.Context, c *envconf.Config) (context.Context, error) {
			log.Println("Finishing tests, cleaning cluster ...")

			// change working directory for Make file
			if err := os.Chdir("../../"); err != nil {
				log.Printf("Unable to set working directory: %s", err)
				return ctx, err
			}

			// uninstall crd
			log.Println("Uninstalling crd...")
			if p := utils.RunCommand(
				`make uninstall ignore-not-found=true`,
			); p.Err() != nil {
				log.Printf("Warning: Failed to uninstall crd: %s: %s", p.Err(), p.Out())
			}

			// undeploy etcd operator
			log.Println("Undeploy etcd controller...")
			if p := utils.RunCommand(
				`make undeploy ignore-not-found=true`,
			); p.Err() != nil {
				log.Printf("Warning: Failed to undeploy controller: %s: %s", p.Err(), p.Out())
			}

			// set working directory test/e2e
			if err := os.Chdir(origWd); err != nil {
				log.Printf("Unable to set working directory: %s", err)
				return ctx, err
			}

			return ctx, nil
		},

		// remove the installed dependencies
		func(ctx context.Context, c *envconf.Config) (context.Context, error) {
			// change working directory for Make file
			if err := os.Chdir("../../"); err != nil {
				log.Printf("Unable to set working directory: %s", err)
				return ctx, err
			}

			log.Println("Removing dependencies...")

			// remove prometheus
			test_utils.UninstallPrometheusOperator()

			// remove cert-manager
			test_utils.UninstallCertManager()

			// set working directory test/e2e
			if err := os.Chdir(origWd); err != nil {
				log.Printf("Unable to set working directory: %s", err)
				return ctx, err
			}

			return ctx, nil
		},

		// Destroy environment
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			var err error

			log.Println("Destroying namespace...")
			ctx, err = envfuncs.DeleteNamespace(namespace)(ctx, cfg)
			if err != nil {
				log.Printf("failed to delete namespace: %s", err)
			}

			log.Println("Destroying cluster...")
			ctx, err = envfuncs.DestroyCluster(kindClusterName)(ctx, cfg)
			if err != nil {
				log.Printf("failed to delete cluster: %s", err)
			}

			return ctx, nil
		},
	)

	// Use Environment.Run to launch the test
	os.Exit(testEnv.Run(m))
}
