// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package constants

const (
	// ExtensionType is the name of the extension type.
	ExtensionType = "shoot-rsyslog-relp"
	// ServiceName is the name of the service.
	ServiceName = "shoot-rsyslog-relp"

	extensionServiceName = "extension-" + ServiceName

	// ManagedResourceName is the name used to describe the managed shoot resources.
	ManagedResourceName = extensionServiceName + "-shoot"
	// ManagedResourceNameConfigCleaner is the name used to describe the manged
	// resource that will clean up the rsyslog config from the shoot nodes.
	ManagedResourceNameConfigCleaner = extensionServiceName + "-configuration-cleaner-shoot"

	// PauseContainerImageName is the name of the pause container image.
	PauseContainerImageName = "pause-container"
	// AlpineImageName is the name of the alpine image.
	AlpineImageName = "alpine"

	// RsyslogCertifcateAuthorityKey is a key in a secret's data which holds the certificate authority used for the tls connection.
	RsyslogCertifcateAuthorityKey = "ca"
	// RsyslogClientCertificateKey is a key in a secret's data which holds the client certificate used for the tls connection.
	RsyslogClientCertificateKey = "crt"
	// RsyslogPrivateKeyKey is a key in a secret's data which holds the private key used for the tls connection.
	RsyslogPrivateKeyKey = "key"
)
