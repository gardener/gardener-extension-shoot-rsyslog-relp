# Auditing Shoot Nodes

## Introduction
As a cluster owner, you might need audit logs on shoot node level. With these audit logs you can track actions on your nodes like privilege escalation, file integrity, process executions and who is the user that performed these actions. Such information is essential for the security of your shoot cluster. Linux operating systems collect such logs via the `auditd` and `journald` daemons. However, these logs can be lost if they are only kept locally on the operating system. You need a reliable way to send them to a remote server where they can be stored for longer time periods and retrieved when necessary.

[Rsyslog](https://www.rsyslog.com/) offers a solution for that. It gathers and process logs from `auditd` and `journald` and then forwards them to a remote server. Moreover, `rsyslog` can make use of the RELP protocol so that logs are sent reliably and no messages are lost.

The `shoot-rsyslog-relp` extension is used to configure `rsyslog` on each `Shoot` node so that the following can take place:
1. `Rsyslog` reads logs from the `auditd` and `journald` sockets.
2. The logs are filtered based on the program name and syslog severity of the message.
3. The logs are enriched with metadata containing the name of the `Project` in which the shoot is created, the name of the `Shoot`, the UID of the `Shoot` and the hostname of the `Node` on which the log event occurred.
4. The enriched logs are sent to the target remote server via the RELP protocol.

The following graph shows a rough outline of how that looks in a shoot cluster:
![rsyslog-logging-architecture](./images/rsyslog-logging-architecture.png)

## Shoot Configuration
The extension is not globally enabled and must be configured per `Shoot` cluster. The `Shoot` specification has to be adapted to include the `shoot-rsyslog-relp` extension configuration which specifies the target server to which logs are forwarded, its port and some optional rsyslog settings described in the examples below.

Below is an example `shoot-rsyslog-relp` extension configuration as part of the `Shoot` spec:

```yaml
kind: Shoot
metadata:
  name: bar
  namespace: garden-foo
...
spec:
  extensions:
  - type: shoot-rsyslog-relp
    providerConfig:
      apiVersion: rsyslog-relp.extensions.gardener.cloud/v1alpha1
      kind: RsyslogRelpConfig
      # Set the target server to which logs are sent. The server must support the RELP protocol.
      target: some.rsyslog-rlep.server
      # Set the port of the target server.
      port: 10250
      # Define rules to select which logs are forwarded to the target server.
      loggingRules:
      - severity: 4
        programNames: ["kubelet", "audisp-syslog"]
      - severity: 1
        programNames: ["audisp-syslog"]
      # Define an interval of 90 seconds at which the current connection is broken and re-established.
      # By default this value is 0 which means that the connection is never broken and re-established.
      rebindInterval: 90
      # Set the timeout for relp sessions to 90 seconds. If set too low, valid sessions may be considered
      # dead and tried to recover.
      timeout: 90
      # Set how often an action is retried before it is considered to have failed.
      # Failed actions discard log messages. Setting `-1` here means that messages are never discarded.
      resumeRetryCount: -1
      # Configures rsyslog to report continuation of action suspension, e.g. when the connection to the target
      # server is broken.
      reportSuspensionContinuation: true
      # Add tls settings if tls should be used to encrypt th connection to the target server.
      tls:
        enabled: true
        # Use `name` authentication mode for the tls connection.
        authMode: name
        # Only allow connections if the server's name is `some.rsyslog-rlep.server`
        permittedPeer: some.rsyslog-rlep.server
        # Reference to the resource which contains certificates used for the tls connection.
        # It must be added to the `.spec.resources` field of the `Shoot`.
        secretReferenceName: rsyslog-relp-tls
  resources:
    # Add the rsyslog-relp-tls secret in the resources field of the shoot spec.
    - name: rsyslog-relp-tls
      resourceRef:
        apiVersion: v1
        kind: Secret
        name: rsyslog-relp-tls-v1
...
```

### Choosing Which Log Messages to Send to the Target Server
The `.loggingRules` field defines rules about which logs should be sent to the target server. When a log is processed by rsyslog it is compared against the list of rules in order. If the program name and the syslog severity of the log messages matches the rule, the message is forwarded to the target server.

Below is an example with a `.loggingRules` section that will only forward logs from the `kubelet` program with syslog severity of 6 or lower and any other program with syslog severity of 2 or lower:

```yaml
apiVersion: rsyslog-relp.extensions.gardener.cloud/v1alpha1
kind: RsyslogRelpConfig
target: localhost
port: 1520
loggingRules:
- severity: 6
  programNames: ["kubelet"]
- severity: 2
```

You can use a minimal `shoot-rsyslog-relp` extension configuration to forward all logs to the target server:

```yaml
apiVersion: rsyslog-relp.extensions.gardener.cloud/v1alpha1
kind: RsyslogRelpConfig
target: some.rsyslog-rlep.server
port: 10250
loggingRules:
- severity: 7
```

### Securing the Communication to the Target Server with TLS
The communication to the target server is not encrypted by default. To enable encryption set the `.tls.enabled` field in the `shoot-rsyslog-relp` extension configuration to `true`. In this case, a `Secret` which contains the TLS certificates used to establish the TLS connection to the server must be created in the same project namespace as your `Shoot`.

An example `Secret` is given below:

```yaml
kind: Secret
apiVersion: v1
metadata:
  name: rsyslog-relp-tls-v1
  namespace: garden-foo
data:
  ca: |
    -----BEGIN BEGIN RSA PRIVATE KEY-----
    ...
    -----END RSA PRIVATE KEY-----
  crt: |
    -----BEGIN BEGIN RSA PRIVATE KEY-----
    ...
    -----END RSA PRIVATE KEY-----
  key: |
    -----BEGIN BEGIN RSA PRIVATE KEY-----
    ...
    -----END RSA PRIVATE KEY-----
```

The `Secret` must be referenced in the `Shoot`'s `.spec.resources` field and the corresponding resource entry must be referenced in the `.tls.secretReferenceName` of the `shoot-rsyslog-relp` extension configuration:

```yaml
kind: Shoot
metadata:
  name: bar
  namespace: garden-foo
...
spec:
  extensions:
  - type: shoot-rsyslog-relp
    providerConfig:
      apiVersion: rsyslog-relp.extensions.gardener.cloud/v1alpha1
      kind: RsyslogRelpConfig
      target: some.rsyslog-rlep.server
      port: 10250
      loggingRules:
      - severity: 7
      tls:
        enabled: true
        secretReferenceName: rsyslog-relp-tls
  resources:
    - name: rsyslog-relp-tls
      resourceRef:
        apiVersion: v1
        kind: Secret
        name: rsyslog-relp-tls-v1
...
```

You can set two additional parameters for the TLS connection: `.tls.authMode` and `tls.permittedPeer`. Refer to the rsyslog documentation for more information on both:
- `.tls.authMode`: https://www.rsyslog.com/doc/v8-stable/configuration/modules/omrelp.html#tls-authmode
- `.tls.permittedPeer`: https://www.rsyslog.com/doc/v8-stable/configuration/modules/omrelp.html#tls-permittedpeer