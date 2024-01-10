// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shoot_test

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/test/framework"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/example/local"
)

const (
	rsyslogRelpEchoServerTag       = "v0.1.0"
	rsyslogRelpEchoServerRepo      = "europe-docker.pkg.dev/gardener-project/releases/gardener/extensions/shoot-rsyslog-relp-echo-server"
	rsyslogRelpEchoServerName      = "rsyslog-relp-echo-server"
	rsyslogRelpEchoServerNamespace = "rsyslog-relp-echo-server"
)

func createRsyslogRelpEchoServer(ctx context.Context, f *framework.ShootFramework) (string, error) {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: rsyslogRelpEchoServerNamespace,
		},
	}
	if err := client.IgnoreAlreadyExists(f.ShootClient.Client().Create(ctx, namespace)); err != nil {
		return "", err
	}

	values := map[string]interface{}{
		"images": map[string]interface{}{
			"rsyslog": fmt.Sprintf("%s:%s", rsyslogRelpEchoServerRepo, rsyslogRelpEchoServerTag),
		},
	}

	if err := f.ShootClient.ChartApplier().ApplyFromEmbeddedFS(ctx, local.Charts, local.RsyslogRelpEchoServerChartPath, rsyslogRelpEchoServerNamespace, rsyslogRelpEchoServerName, kubernetes.Values(values), kubernetes.ForceNamespace); err != nil {
		return "", err
	}

	if err := f.WaitUntilDeploymentIsReady(ctx, rsyslogRelpEchoServerName, rsyslogRelpEchoServerNamespace, f.ShootClient); err != nil {
		return "", err
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: rsyslogRelpEchoServerNamespace,
			Name:      rsyslogRelpEchoServerName,
		},
	}

	if err := f.ShootClient.Client().Get(ctx, client.ObjectKeyFromObject(service), service); err != nil {
		return "", err
	}
	if len(service.Spec.ClusterIPs) == 0 {
		return "", fmt.Errorf("service %s does not have a ClusterIP assigned", client.ObjectKeyFromObject(service).String())
	}
	return service.Spec.ClusterIPs[0], nil
}

func deleteRsyslogRelpEchoServer(ctx context.Context, f *framework.ShootFramework) error {
	return f.ShootClient.ChartApplier().DeleteFromEmbeddedFS(ctx, local.Charts, local.RsyslogRelpEchoServerChartPath, rsyslogRelpEchoServerNamespace, rsyslogRelpEchoServerName)
}
