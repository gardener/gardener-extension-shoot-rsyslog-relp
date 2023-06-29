# Configure rsyslog on shoot nodes to forward system logs to a remote server via the RELP protocol

## Introduction
[Rsyslog](https://www.rsyslog.com/) is a powerful log processing tool that is installed on `Shoot` nodes running `gardenlinux` after version `934.7` and `suse-chost` after version `15.4.20230510`. This extension can be used to configure rsyslog on the `Shoot` nodes so that it collects system logs and audit logs and forwards them to a log aggregation server via the [relp](https://www.rsyslog.com/download/files/windows-agent-manual/glossaryofterms/relp.html) protocol.

## Shoot Configuration
The extension is not globally enabled and must be configured per `Shoot` cluster. The `Shoot` specification has to be adapted to include the `shoot-rsyslog-relp` extension configuration.

The configuration specifies the target server to which logs are forwarded, its port and some optional rsyslog settings. For more information on the available settings check the [official rsyslog relp documentation](https://www.rsyslog.com/doc/v8-stable/configuration/modules/imrelp.html).

When tls is enabled via the `tls.enabled` field, the communication to the target server is encrypted. This requires users provide a `Secret` resource, which contains the required tls certificates, in the `Shoot`'s `resources` array and refer to it via the `tls.secretReferenceName` field.

The `filters` field can be used to limit the amount of logs that are forwarded to the target server. When it is empty, all syslog and audit events are forwarded to the target server.

Check the examples below for more information.

Example configuration for the extension:

```yaml
kind: Shoot
...
spec:
  extensions:
  - type: shoot-rsyslog-relp
    providerConfig:
      apiVersion: rsyslog-relp.extensions.gardener.cloud/v1alpha1
      kind: RsyslogRelpConfig
      target: localhost
      port: 1520
      tls:
        enabled: true
        authMode: name # optional
        permittedPeer: peer # optional
        secretReferenceName: rsyslog-relp-tls
      filters: |
        if $programname in ["kubelet"] then {
          call defaultruleset
        }
      rebindInterval: 90 # optional
      timeout: 90 # optional
      resumeRetryCount: true # optionaal
      reportSuspensionContinuation: true # optional
  resources:
    - name: rsyslog-relp-tls
      resourceRef:
        apiVersion: v1
        kind: Secret
        name: rsyslog-relp-tls
...
```

Example `Secret` resource which contains the tls certificates required for encrypting the communication to the target server:

```yaml
kind: Secret
apiVersion: v1
metadata:
  name: rsyslog-credentials
  namespace: garden-foo
data:
  key: |
    -----BEGIN BEGIN RSA PRIVATE KEY-----
    ...
    -----END RSA PRIVATE KEY-----
  crt: |
    -----BEGIN BEGIN RSA PRIVATE KEY-----
    ...
    -----END RSA PRIVATE KEY-----
  ca: |
    -----BEGIN BEGIN RSA PRIVATE KEY-----
    ...
    -----END RSA PRIVATE KEY-----
```