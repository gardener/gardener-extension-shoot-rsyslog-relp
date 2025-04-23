// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	_ "embed"
	"errors"
	"time"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/utils/retry"
	testutils "github.com/gardener/gardener/pkg/utils/test"
	"github.com/gardener/gardener/test/framework"
	"github.com/go-logr/logr"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	clientcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetEchoServerPodInterfaceAndName returns the clientcorev1.PodInterface and the name of the pod
// for the rsyslog-relp-echo-server deployment.
func GetEchoServerPodInterfaceAndName(ctx context.Context, c kubernetes.Interface) (clientcorev1.PodInterface, string, error) {
	podIf := c.Kubernetes().CoreV1().Pods("rsyslog-relp-echo-server")

	pods := &corev1.PodList{}
	if err := c.Client().List(ctx, pods, client.InNamespace("rsyslog-relp-echo-server")); err != nil {
		return nil, "", err
	}
	if len(pods.Items) == 0 {
		return nil, "", errors.New("could not find any rsyslog-relp-echo-server pods")
	}

	return podIf, pods.Items[0].Name, nil
}

// ForEachNode executes the given function for each node retrieved with the given client.
func ForEachNode(ctx context.Context, c kubernetes.Interface, fn func(ctx context.Context, node *corev1.Node)) {
	nodes := &corev1.NodeList{}
	ExpectWithOffset(1, c.Client().List(ctx, nodes)).To(Succeed())

	for _, node := range nodes.Items {
		fn(ctx, &node)
	}
}

// ExecCommand uses the given RootPodExecutor to execute the given command.
func ExecCommand(ctx context.Context, log logr.Logger, podExecutor framework.RootPodExecutor, command ...string) (response []byte, err error) {
	err = retry.Until(ctx, 5*time.Second, func(ctx context.Context) (bool, error) {
		response, err = podExecutor.Execute(ctx, command...)
		if err != nil {
			log.Error(err, "Error exec'ing into pod")
			return retry.MinorError(err)
		}

		return retry.Ok()
	})
	return
}

// CreateResourcesFromFile creates the objects from filePath with a given namespace name
func CreateResourcesFromFile(ctx context.Context, client client.Client, namespaceName string, filePath string) ([]client.Object, error) {
	resources, err := testutils.ReadTestResources(client.Scheme(), namespaceName, filePath)
	if err != nil {
		return nil, err
	}
	for _, obj := range resources {
		if err = client.Create(ctx, obj); err != nil {
			return nil, err
		}
	}
	return resources, nil
}
