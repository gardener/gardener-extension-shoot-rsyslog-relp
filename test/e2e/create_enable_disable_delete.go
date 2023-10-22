// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"context"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	e2e "github.com/gardener/gardener/test/e2e/gardener"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Shoot Rsyslog Relp Extension Tests", func() {
	test := func(shoot *gardencorev1beta1.Shoot, shootMutateFn func(shoot *gardencorev1beta1.Shoot) error, gardenClusterResources ...client.Object) {
		f := defaultShootCreationFramework()
		f.Shoot = shoot

		It("Create Shoot, enable shoot-rsyslog-relp extension then disable it and delete Shoot", Offset(1), func() {
			ctx, cancel := context.WithTimeout(parentCtx, 20*time.Minute)
			DeferCleanup(cancel)

			By("Crerate additional garden cluster resources")
			for _, obj := range gardenClusterResources {
				obj.SetNamespace(f.ProjectNamespace)
				Expect(f.GardenClient.Client().Create(ctx, obj)).To(Succeed())
				f.Logger.Info("Created resource", "resource", client.ObjectKeyFromObject(obj))
			}

			By("Create Shoot")
			Expect(f.CreateShootAndWaitForCreation(ctx, false)).To(Succeed())
			f.Verify()

			ctx, cancel = context.WithTimeout(parentCtx, 1*time.Minute)
			DeferCleanup(cancel)

			By("Create NetworkPolicy to allow traffic from Shoot nodes to the rsyslog-relp echo server")
			Expect(createNetworkPolicyForEchoServer(ctx, f.ShootFramework.SeedClient, f.ShootFramework.ShootSeedNamespace())).To(Succeed())

			By("Install rsyslog-relp unit on Shoot nodes")
			Expect(execInShootNode(ctx, f.ShootFramework.SeedClient, f.Logger, f.ShootFramework.ShootSeedNamespace(), "apt-get update && apt-get install -y rsyslog-relp")).To(Succeed())

			By("Enable the shoot-rsyslog-relp extension")
			ctx, cancel = context.WithTimeout(parentCtx, 15*time.Minute)
			DeferCleanup(cancel)
			Expect(f.UpdateShoot(ctx, f.Shoot, shootMutateFn)).To(Succeed())

			By("Verify shoot-rsyslog-relp works")
			ctx, cancel = context.WithTimeout(parentCtx, 5*time.Minute)
			DeferCleanup(cancel)

			verifier, err := newVerifier(ctx, f.Logger, f.ShootFramework.SeedClient, f.ShootFramework.ShootSeedNamespace(), "local", f.Shoot.Name, string(f.Shoot.UID))
			Expect(err).NotTo(HaveOccurred())

			verifier.verifyThatLogsAreSentToEchoServer(ctx, "test-program", "1", "this should get sent to echo server")
			verifier.verifyThatLogsAreNotSentToEchoServer(ctx, "other-program", "1", "this should not get sent to echo server")
			verifier.verifyThatLogsAreNotSentToEchoServer(ctx, "test-program", "3", "this should not get sent to echo server")

			By("Disable the shoot-rsyslog-relp extension")
			ctx, cancel = context.WithTimeout(parentCtx, 15*time.Minute)
			DeferCleanup(cancel)
			Expect(f.UpdateShoot(ctx, f.Shoot, func(shoot *gardencorev1beta1.Shoot) error {
				for i, extension := range shoot.Spec.Extensions {
					if extension.Type == "shoot-rsyslog-relp" {
						shoot.Spec.Extensions = append(shoot.Spec.Extensions[:i], shoot.Spec.Extensions[i+1:]...)
					}
				}
				return nil
			})).To(Succeed())

			By("Verify that shoot-rsyslog-relp extension is disabled")
			ctx, cancel = context.WithTimeout(parentCtx, 5*time.Minute)
			DeferCleanup(cancel)
			verifier.verifyThatLogsAreNotSentToEchoServer(ctx, "test-program", "1", "this should not get sent to echo server")

			By("Delete Shoot")
			ctx, cancel = context.WithTimeout(parentCtx, 15*time.Minute)
			DeferCleanup(cancel)
			Expect(f.DeleteShootAndWaitForDeletion(ctx, f.Shoot)).To(Succeed())

			By("Delete additional garden cluster resources")
			for _, obj := range gardenClusterResources {
				Expect(f.GardenClient.Client().Delete(ctx, obj)).To(Succeed())
				f.Logger.Info("Deleted resource", "resource", client.ObjectKeyFromObject(obj))
			}
		})
	}

	Context("shoot-rsyslog-relp extension with tls disabled", Label("tls-disabled"), func() {
		test(e2e.DefaultShoot("e2e-rslog-relp"), func(shoot *gardencorev1beta1.Shoot) error {
			shoot.Spec.Extensions = append(shoot.Spec.Extensions, shootRsyslogRelpExtension())
			return nil
		})
	})

	Context("shoot-rsyslog-relp extension with tls enabled", Label("tls-enabled"), func() {
		test(
			e2e.DefaultShoot("e2e-rslog-tls"),
			func(shoot *gardencorev1beta1.Shoot) error {
				shoot.Spec.Extensions = append(shoot.Spec.Extensions, shootRsyslogRelpExtension(withPort(443), withTLSWithSecretRefName("rsyslog-relp-tls")))
				shoot.Spec.Resources = append(shoot.Spec.Resources, gardencorev1beta1.NamedResourceReference{
					Name: "rsyslog-relp-tls",
					ResourceRef: autoscalingv1.CrossVersionObjectReference{
						Kind:       "Secret",
						APIVersion: "v1",
						Name:       "rsyslog-relp-tls",
					},
				})
				return nil
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "rsyslog-relp-tls",
				},
				Data: map[string][]byte{
					"ca": []byte(`-----BEGIN CERTIFICATE-----
MIIDMDCCAhigAwIBAgIUDugzjXoDy5VaT4Nc3z2CQISLUOAwDQYJKoZIhvcNAQEL
BQAwMDELMAkGA1UEBhMCREUxDzANBgNVBAoTBlNBUCBTRTEQMA4GA1UEAxMHcnN5
c2xvZzAeFw0yMzEwMjIxNjM1MjNaFw0zMzEwMTkxNjM1MjZaMDAxCzAJBgNVBAYT
AkRFMQ8wDQYDVQQKEwZTQVAgU0UxEDAOBgNVBAMTB3JzeXNsb2cwggEiMA0GCSqG
SIb3DQEBAQUAA4IBDwAwggEKAoIBAQDZqRwYTUhx+KkgS2p26oLpG9ppQjYptIq4
WQ1z1aF7vdr1l664qpfpzIPOYXiBLDLjeLtyCt5po3E2/jdFYdKy/YRsG+HIqidC
SW+WHmt+AOb5ooZ+vJVVlLjHWIw7kFx5aLR29FJtHaeblC23semON7GZACWpoC/c
t/SK3VLJbiMHxF0/bZjjmXl8TeLttHqfZBmeAUDfxm2Y5+mflvNBhHQDS47qFsU8
6bi+GWeNOV8IlVDUeSt8MmDlzRLEEz8BdG0DU/l9RzbFRq77yWwI9hHKnSrvAEU/
HNUPqFAUVEyAARjGv6OOS5BPvjEfo7VXKk1QXohC3XvQ0tA6l5grAgMBAAGjQjBA
MA8GA1UdEwEB/wQFMAMBAf8wDgYDVR0PAQH/BAQDAgKEMB0GA1UdDgQWBBQZs6ku
aSMBH8j7NiqhbkuR4quShjANBgkqhkiG9w0BAQsFAAOCAQEAtDTZkgMF92ayryaQ
Kl32x1cen1DJxGx8XsROhTkJKNNfUtjfTn5xX+VapjdOYUgOX22ORqWr5vD+5s/x
/bQC1FNxTby3wvd/BmihpvbsWGmQl6R8EhKDOjqRgi7bIQUb9aFXqwG6pO3zLbvl
Igt7Puwt7nVXr7tsfVp0LSclrUwUy7CbV0rt+7Wi1g9kOpLFFn2q6ZAL7ZP88pUO
nHE0RyYEqft2BWPWZ1DhiM/dSpIcaYJ1ZrhTyEin6uh2zA5i+56sTvayHu5AczDX
d83fMx4+/g/dwlEZpohCZD7DemdVLiDjTLvVqdJiTLn+hHdQmQMk06SGNI/Sn6vw
Rf+TEQ==
-----END CERTIFICATE-----
`),
					"crt": []byte(`-----BEGIN CERTIFICATE-----
MIIDijCCAnKgAwIBAgIUdeMX++rb9+oRWdMd3GsKC62dUmwwDQYJKoZIhvcNAQEL
BQAwMDELMAkGA1UEBhMCREUxDzANBgNVBAoTBlNBUCBTRTEQMA4GA1UEAxMHcnN5
c2xvZzAeFw0yMzEwMjIxNjM4NDdaFw0zMzEwMTkxNjM4NTBaMDAxCzAJBgNVBAYT
AkRFMQ8wDQYDVQQKEwZTQVAgU0UxEDAOBgNVBAMTB3JzeXNsb2cwggEiMA0GCSqG
SIb3DQEBAQUAA4IBDwAwggEKAoIBAQDFcnK8sfo6TcniqjI9j30lVpsYwCHlfKst
HYmA5OUtd6Wks3t9m7AEyxLKMe2DSqV1etvDtOIHbW4OllA/e1G9hqiBQnUIMkP3
QLseHKNSnTKzm4nUTyVx1dBqnp6KDa77ChWpxqDyMzKa6So4rId9FIsDgxEuFSWo
wHUbcGdmvhYuqticFqltLi6iK4FWEwyMF/rZ/RXHRRZYtC1LB6/b46Q0Ljz7kYiM
Iu6bQ+H9FBDcv/7rGZCB/R+iJ4NlJnItSpYf9yrbyc3Y+pRPutl1SPOcronuG36r
1JHLaMghAxIuPSImiac8lq+dXoXsDXp1KoD8WIsAzR/OpaT7jVNrAgMBAAGjgZsw
gZgwDAYDVR0TAQH/BAIwADAdBgNVHSUEFjAUBggrBgEFBQcDAgYIKwYBBQUHAwEw
GQYDVR0RBBIwEIIOcnN5c2xvZy1zZXJ2ZXIwDgYDVR0PAQH/BAQDAgWgMB0GA1Ud
DgQWBBRvRtO8ENgH2wqw4PHuhLmLOkSVtDAfBgNVHSMEGDAWgBQZs6kuaSMBH8j7
NiqhbkuR4quShjANBgkqhkiG9w0BAQsFAAOCAQEAJqJdtkBDV27TjY2vg7eaDaYt
eSryD0TSA9uUF7pLCCyqsPhWk2zLvFgUbrJw0sKGUYf1I/A4Z3G25i6mBl5AGhBE
TVbf1h2uq02PJCmlitb6X0ZfqjJuiTC+3mE2c3IPMH7QdMAVAEghaepk/aBFtScD
MGfzVJ39PvCt4pcn9sYsBiXFMioAX8FMhtmjAbkZwbrpPE072snv1TEtk11yzlyn
Jd49/7H/tCKgW1NnsoDcO4SBr7PaDLG0pd+9suqCpmO9JM/XGYPpX7qxzHArk/9o
yzDgAX2amQ4/TER3Ylb4Fo94nNpZOLzwqBIIwXgWGooPlY5uaZPoufy18ceRWA==
-----END CERTIFICATE-----
`),
					"key": []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAxXJyvLH6Ok3J4qoyPY99JVabGMAh5XyrLR2JgOTlLXelpLN7
fZuwBMsSyjHtg0qldXrbw7TiB21uDpZQP3tRvYaogUJ1CDJD90C7HhyjUp0ys5uJ
1E8lcdXQap6eig2u+woVqcag8jMymukqOKyHfRSLA4MRLhUlqMB1G3BnZr4WLqrY
nBapbS4uoiuBVhMMjBf62f0Vx0UWWLQtSwev2+OkNC48+5GIjCLum0Ph/RQQ3L/+
6xmQgf0foieDZSZyLUqWH/cq28nN2PqUT7rZdUjznK6J7ht+q9SRy2jIIQMSLj0i
JomnPJavnV6F7A16dSqA/FiLAM0fzqWk+41TawIDAQABAoIBAQCU8db33VKj9NZc
tIMdyUZgikqJizaGxVrjt6pon0L6340HB5YalP1dQEu2V5+SMRdL3hg2NBdl/vjM
7DsxCDgLPq+Sgq2CN1jqBdyhxHy373m135lDnUjj7KVCKNHz1oqvOVZKMlprGpAM
J+P/yLaUdpC/X3nwR2eXO0ecIVj/OQZt8nReRUya/e13BuRNsJ5mo3DFkRQYJtgi
ZBxVVwPCuEYpmLMVt5Wf8jAQVuOR877hXakxD2bkFJIKcAd8buewHscKyiz5nHJv
hVVGfuQzM7EHH37OT5j3d1KKHSSjmL7jHV+drghSm44eOffrzk/d86G9XVv21hbT
RqNb8wfhAoGBAOeIWL9GfYvJIG52XC3TBbB1IJTM2PecMuE+IVCZAvdb1ZaFgwRC
VbEmaFJ87OcVW8uVu20KZpjkUVqbVRbtfMdtL9sHsmj3TH2MH7dWsy96CGx/2+Xa
uHVtoTzV8tqzM2GDcF0JegeuDEaK1NEjXJX4CWN65NV78kBLZtNS2HBJAoGBANpP
/DDhjmm/oUvH6P4sEF966BL/g0jcgKgy0RcJLmuVpde8iBbQhZe/1KVSl9+xzCvb
ZtFztIgEmIGPTwWi42yODKvSQ2mLC1IKTk1J9DtVR+Z9qbEqz5smHCogrfjxAmnM
jNC9AdbAzQowKsn5bVGU8PrzmDgHhX8jpe82kQ4TAoGAJnDj0zYf6BKHmO971Hvh
yO9ZbnsoVswPQohvPZN6A5myt6AJJa7hzVzEG1X0e1V3fTCqAqukZyQZQcLieMEL
Y40EUghQHc9ZWsrmBSmW7H4FYgZEe0A6OfzutUwMWzU/haQuBrRpF1dVYGzycpq9
Z4TcAjFIRw2iJfye4N0zZEkCgYA4H3fl2RaTeQAuSyZKsWlEIoSm3akSgh1RID9A
fMvCPKZ137Hcq56sdFRma+U/TKYAYFb+YZB3pzbNl9noyQdOUPZQ9az+5Q/z91JJ
7EktN69UQdnuAeN9Lz7uVZhj9xF3wW4x+2UNoGMVy2w0oDrKTk/lM9peDRD0rmVq
Kc0AoQKBgQCJ84ULqiFXdL5qyHn12tbGwvxIRPP26E8eCC5gLA7ndxFG9//L8bre
BEwthU+Nc5SkaaTB653V7xOcJcaP0nxKUyz8zYL4DxGcJKweDpEwDB2wBY0+57qX
qZ1g6Bb+fTjXinD9Lf5RbkvzTyiOMTv8sT/dHfz+/ooV6pSa4DGm3g==
-----END RSA PRIVATE KEY-----
`),
				},
			})
	})
})
