# [Gardener Extension to configure rsyslog with relp module](https://gardener.cloud)

[![reuse compliant](https://reuse.software/badge/reuse-compliant.svg)](https://reuse.software/)

Project Gardener implements the automated management and operation of [Kubernetes](https://kubernetes.io/) clusters as a service.
Its main principle is to leverage Kubernetes concepts for all of its tasks.

Recently, most of the vendor specific logic has been developed [in-tree](https://github.com/gardener/gardener).
However, the project has grown to a size where it is very hard to extend, maintain, and test.
With [GEP-1](https://github.com/gardener/gardener/blob/master/docs/proposals/01-extensibility.md) we have proposed how the architecture can be changed in a way to support external controllers that contain their very own vendor specifics.
This way, we can keep Gardener core clean and independent.

This controller implements Gardener's extension contract for the `shoot-rsyslog-relp` extension.

An example for a `ControllerRegistration` resource that can be used to register this controller to Gardener can be found [here](example/controller-registration.yaml).

Please find more information regarding the extensibility concepts and a detailed proposal [here](https://github.com/gardener/gardener/blob/master/docs/proposals/01-extensibility.md).

## Configuration

Example configuration for this extension controller:

```yaml
apiVersion: rsyslog-relp.extensions.config.gardener.cloud/v1alpha1
kind: RsyslogRelpConfig
target: relp-server.foo.bar
port: 1520
tls:
  enabled: true
  authMode: name
  permittedPeer:
  - "some.rsyslog-rlep.server"
  secretReferenceName: rsyslog-relep-tls
```

## Extension Resources

Example extension resource:

```yaml
apiVersion: extensions.gardener.cloud/v1alpha1
kind: Extension
metadata:
  name: extension-shoot-rsyslog-relp
  namespace: shoot--project--abc
spec:
  type: shoot-rsyslog-relp
```

When an extension resource is reconciled, the extension controller will create a daemonset in the `Shoot` cluster which will configure rsyslog that is already installed on `gardenlinux` and `suse-chost` nodes. It will take care of provisioning any secrets and custom filters required by rsyslog on the nodes. If the `Shoot` workers do not run `gardenlinux` or `suse-chost` the daemonset is still created, however it will have no effect. The daemonset will also configure auditd rules so that audit logs can be forwarded to rsyslog.

Please note, this extension controller relies on the [Gardener-Resource-Manager](https://github.com/gardener/gardener/blob/master/docs/concepts/resource-manager.md) to deploy k8s resources to seed and shoot clusters.

## How to start using or developing this extension controller locally

Check the [Running Rsyslog Relp Extension Locally](./docs/development/getting-started.md) to get started.

## Feedback and Support

Feedback and contributions are always welcome. Please report bugs or suggestions as [GitHub issues](https://github.com/gardener/gardener-extension-shoot-rsyslog-relp/issues) or join our [Slack channel #gardener](https://kubernetes.slack.com/messages/gardener) (please invite yourself to the Kubernetes workspace [here](http://slack.k8s.io)).

## Learn more!

Please find further resources about out project here:

* [Our landing page gardener.cloud](https://gardener.cloud/)
* ["Gardener, the Kubernetes Botanist" blog on kubernetes.io](https://kubernetes.io/blog/2018/05/17/gardener/)
* ["Gardener Project Update" blog on kubernetes.io](https://kubernetes.io/blog/2019/12/02/gardener-project-update/)
* [GEP-1 (Gardener Enhancement Proposal) on extensibility](https://github.com/gardener/gardener/blob/master/docs/proposals/01-extensibility.md)
* [Extensibility API documentation](https://github.com/gardener/gardener/tree/master/docs/extensions)
* [Gardener Extensions Golang library](https://godoc.org/github.com/gardener/gardener/extensions/pkg)
* [Gardener API Reference](https://gardener.cloud/api-reference/)
