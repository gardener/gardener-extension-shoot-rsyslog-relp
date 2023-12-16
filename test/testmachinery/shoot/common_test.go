// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shoot_test

import (
	"context"
	"fmt"
	"net"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/test/framework"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/example/local"
)

const (
	rsyslogRelpEchoServerTag       = "v0.1.0"
	rsyslogRelpEchoServerRepo      = "eu.gcr.io/gardener-project/gardener/extensions/shoot-rsyslog-relp-echo-server"
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

	clusterIP, err := findFreeClusterIPAddress(ctx, f, namespace.Name)
	if err != nil {
		return "", err
	}

	values := map[string]interface{}{
		"images": map[string]interface{}{
			"rsyslog": fmt.Sprintf("%s:%s", rsyslogRelpEchoServerRepo, rsyslogRelpEchoServerTag),
		},
		"service": map[string]interface{}{
			"clusterIP": clusterIP,
		},
	}

	if err := f.ShootClient.ChartApplier().ApplyFromEmbeddedFS(ctx, local.Charts, local.RsyslogRelpEchoServerChartPath, rsyslogRelpEchoServerNamespace, rsyslogRelpEchoServerName, kubernetes.Values(values), kubernetes.ForceNamespace); err != nil {
		return "", err
	}

	if err := f.WaitUntilDeploymentIsReady(ctx, rsyslogRelpEchoServerName, rsyslogRelpEchoServerNamespace, f.ShootClient); err != nil {
		return "", err
	}

	return clusterIP, nil
}

func deleteRsyslogRelpEchoServer(ctx context.Context, f *framework.ShootFramework) error {
	return f.ShootClient.ChartApplier().DeleteFromEmbeddedFS(ctx, local.Charts, local.RsyslogRelpEchoServerChartPath, rsyslogRelpEchoServerNamespace, rsyslogRelpEchoServerName)
}

func findFreeClusterIPAddress(ctx context.Context, f *framework.ShootFramework, namespace string) (string, error) {
	existingService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      rsyslogRelpEchoServerName,
		},
	}

	if err := f.ShootClient.Client().Get(ctx, client.ObjectKeyFromObject(existingService), existingService); err == nil {
		return existingService.Spec.ClusterIP, nil
	} else if !errors.IsNotFound(err) {
		return "", err
	}

	clusterIPRange := *f.Shoot.Spec.Networking.Services

	serviceList := &corev1.ServiceList{}
	if err := f.ShootClient.Client().List(ctx, serviceList); err != nil {
		return "", err
	}

	clusterIPsInUse := make(map[string]struct{}, len(serviceList.Items))
	for _, service := range serviceList.Items {
		clusterIPsInUse[service.Spec.ClusterIP] = struct{}{}
	}

	ip, ipnet, err := net.ParseCIDR(clusterIPRange)
	if err != nil {
		return "", err
	}

	inc := func(ip net.IP) {
		for j := len(ip) - 1; j >= 0; j-- {
			ip[j]++
			if ip[j] > 0 {
				break
			}
		}
	}

	// Increment the IP once at the start so that we skip the network address.
	for inc(ip); ipnet.Contains(ip); inc(ip) {
		if _, found := clusterIPsInUse[ip.String()]; !found {
			return ip.String(), nil
		}
	}

	return "", nil
}
