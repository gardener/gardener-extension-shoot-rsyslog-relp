// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package operatingsystemconfig_test

import (
	"bytes"
	"encoding/json"
	"text/template"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/webhook/operatingsystemconfig/webhooktest"
)

type configData struct {
	// ProjectName to fill in template for config.
	ProjectName string
	// Target to fill in template for config.
	Target string
	// Port to fill in template for config.
	Port int
}

var _ = Describe("Webhook tests", func() {
	var (
		osc        *extensionsv1alpha1.OperatingSystemConfig
		config     *rsyslog.RsyslogRelpConfig
		configData configData
	)

	JustBeforeEach(func() {
		By("Create Extension")
		extensionProviderConfigJSON, err := json.Marshal(config)
		Expect(err).NotTo(HaveOccurred())
		extension := &extensionsv1alpha1.Extension{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "shoot-rsyslog-relp",
				Namespace: cluster.Name,
			},
			Spec: extensionsv1alpha1.ExtensionSpec{
				DefaultSpec: extensionsv1alpha1.DefaultSpec{
					ProviderConfig: &runtime.RawExtension{
						Raw: extensionProviderConfigJSON,
					},
				},
			},
		}
		Expect(testClient.Create(ctx, extension)).To(Succeed())
		log.Info("Created Extension", "cluster", client.ObjectKeyFromObject(extension))
		DeferCleanup(func() {
			By("Delete Extension")
			Expect(client.IgnoreNotFound(testClient.Delete(ctx, extension))).To(Succeed())
		})

		osc = &extensionsv1alpha1.OperatingSystemConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-osc",
				Namespace: testNamespace.Name,
			},
			Spec: extensionsv1alpha1.OperatingSystemConfigSpec{
				Purpose: extensionsv1alpha1.OperatingSystemConfigPurposeReconcile,
				Units: []extensionsv1alpha1.Unit{
					{
						Name: "foo",
						DropIns: []extensionsv1alpha1.DropIn{
							{
								Name:    "drop1",
								Content: "data1",
							},
						},
						FilePaths: []string{"/foo-bar-file"},
					},
				},
				Files: []extensionsv1alpha1.File{
					{
						Path: "foo/bar",
						Content: extensionsv1alpha1.FileContent{
							Inline: &extensionsv1alpha1.FileContentInline{
								Encoding: "b64",
								Data:     "some-data",
							},
						},
					},
					{
						Path: "/foo-bar-file",
						Content: extensionsv1alpha1.FileContent{
							ImageRef: &extensionsv1alpha1.FileContentImageRef{
								Image:           "foo-image:bar-tag",
								FilePathInImage: "/foo-bar-file",
							},
						},
					},
				},
			},
		}
		configData.ProjectName = testNamespace.Name
		By("Create OSC")
		// Eventually is used because webhook may still not be ready to receive traffic
		Eventually(func() error {
			return testClient.Create(ctx, osc)
		}).Should(Succeed())
		DeferCleanup(func() {
			By("Delete OSC")
			Expect(client.IgnoreNotFound(testClient.Delete(ctx, osc))).To(Succeed())
		})

	})

	Context("Default Behaviour of extension with a default configuration", func() {
		BeforeEach(func() {
			config = &rsyslog.RsyslogRelpConfig{
				Target: "localhost",
				Port:   1520,
			}
			configData.Target = "localhost"
			configData.Port = 1520
		})

		It("should have all expected files attached", func() {
			expectedFiles := []extensionsv1alpha1.File{
				{
					Path: "foo/bar",
					Content: extensionsv1alpha1.FileContent{
						Inline: &extensionsv1alpha1.FileContentInline{
							Encoding: "b64",
							Data:     "some-data",
						},
					},
				},
				{
					Path: "/foo-bar-file",
					Content: extensionsv1alpha1.FileContent{
						ImageRef: &extensionsv1alpha1.FileContentImageRef{
							Image:           "foo-image:bar-tag",
							FilePathInImage: "/foo-bar-file",
						},
					},
				},
			}
			tmpl, err := template.New("config").Parse(string(webhooktest.GetSimpleRsyslogConfig()))
			Expect(err).NotTo(HaveOccurred())
			var simpleRsyslogConfig bytes.Buffer
			Expect(tmpl.Execute(&simpleRsyslogConfig, configData)).To(Succeed())

			expectedFiles = append(expectedFiles, webhooktest.GetRsyslogFiles(simpleRsyslogConfig.Bytes(), true)...)
			expectedFiles = append(expectedFiles, webhooktest.GetAuditRulesFiles(true)...)
			Expect(osc.Spec.Files).To(HaveLen(len(expectedFiles)))
			Expect(osc.Spec.Files).To(ConsistOf(expectedFiles))
		})

		It("should have all expected units attached", func() {
			expectedUnits := []extensionsv1alpha1.Unit{
				{
					Name: "foo",
					DropIns: []extensionsv1alpha1.DropIn{
						{
							Name:    "drop1",
							Content: "data1",
						},
					},
					FilePaths: []string{"/foo-bar-file"},
				},
			}
			expectedUnits = append(expectedUnits, []extensionsv1alpha1.Unit{webhooktest.GetRsyslogConfiguratorUnit(true)}...)
			Expect(osc.Spec.Units).To(HaveLen(len(expectedUnits)))
			Expect(osc.Spec.Units).To(ConsistOf(expectedUnits))
		})
	})
})
