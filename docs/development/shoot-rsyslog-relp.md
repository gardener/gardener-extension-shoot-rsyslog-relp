# Developer Docs for Gardener Shoot Rsyslog Relp Extension

This document outlines how the Shoot reconciliation and deletion work for a Shoot with the shoot-rsyslog-relp extension enabled.

## Shoot reconciliation

This section outlines how the reconciliation works for a Shoot with the shoot-rsyslog-relp extension enabled.

#### Extension enablement/reconciliation

This section outlines how the extension enablement/reconciliation works, e.g the extension has been added to the Shoot spec.

1. As part of the Shoot reconciliation flow, gardenlet deploys the [Extension](https://github.com/gardener/gardener/blob/v1.82.0/docs/extensions/extension.md) resource.
1. The shoot-rsyslog-relp extension reconciles the Extension resource. [pkg/controller/lifecycle/actuator.go](../../pkg/controller/extension/actuator.go) contains the implementation of the [extension.Actuator](https://github.com/gardener/gardener/blob/v1.82.0/extensions/pkg/controller/extension/actuator.go) interface. The reconciliation of an Extension of type `shoot-rsyslog-relp` only deploys the necessary monitoring configuration - the `shoot-rsyslog-relp-prometheus` ConfigMap which contains the definitions for: scraping metrics by prometheus, alerting rules and the plutono dashboard for the Rsyslog component.
1. As part of the Shoot reconciliation flow, gardenlet deploys the OperatingSystemConfig resource.
1. The shoot-rsyslog-relp extension serves a webhook that mutates the OperatingSystemConfig resource for Shoots having the shoot-rsyslog-relp extension enabled (the corresponding namespace gets labeled by gardenlet with `extensions.gardener.cloud/shoot-rsyslog-relp=true`). [pkg/webhook/operatingsystemconfig/ensurer.go](../../pkg/webhook/operatingsystemconfig/ensurer.go) contains implementation of the [genericmutator.Ensurer](https://github.com/gardener/gardener/blob/v1.82.0/extensions/pkg/webhook/controlplane/genericmutator/mutator.go) interface.
    1. The webhook renders the [60-audit.conf.tpl](../../pkg/webhook/operatingsystemconfig/resources/templates/scripts/configure-rsyslog.tpl.sh) template script and appends it to the OperatingSystemConfig files. When rendering the template, the configuration of the shoot-rsyslog-relp extension is used to fill in the required template values. The file is installed as `/var/lib/rsyslog-relp-configurator/rsyslog.d/60-audit.conf` on the host os.
    1. The webhook appends the [audit rules](../../pkg/webhook/operatingsystemconfig/resources/auditrules/) to the OperatingSystemConfig. The files are installed under `/var/lib/rsyslog-relp-configurator/rules.d` on the host os.
    1. The webhook renders the [configure-rsyslog.tpl.sh](../../pkg/webhook/operatingsystemconfig/resources/templates/scripts/configure-rsyslog.tpl.sh) script and appends it to the OperatingSystemConfig files. This script is installed as `/var/lib/rsyslog-relp-configurator/configure-rsyslog.sh` on the host os. It keeps the configuration of the `rsyslog` systemd service up-to-date by copying `/var/lib/rsyslog-relp-configurator/rsyslog.d/60-audit.conf` to `/etc/rsyslog.d/60-audit.conf`, if `/etc/rsyslog.d/60-audit.conf` does not exist or the files differ. The script also takes care of syncing the audit rules in `/etc/audit/rules.d` with the ones installed in `/var/lib/rsyslog-relp-configurator/rules.d` and restarts the auditd systemd service if necessary.
    1. The webhook renders the [process-rsyslog-pstats.tpl.sh](../../pkg/webhook/operatingsystemconfig/resources/templates/scripts/process-rsyslog-pstats.tpl.sh) and appends it to the OperatingSystemConfig files. This script receives metrics from the `rsyslog` process, transforms them and writes them to `/var/lib/node-exporter/textfile-collector/rsyslog_pstats.prom` so that they can be collected by the `node-exporter`.
    1. As part of the Shoot reconciliation, before the shoot-rsyslog-relp extension is deployed, gardenlet copies all Secret and ConfigMap resources referenced in `.spec.resources[]` to the Shoot's control plane namespace on the Seed. 
    When the `.tls.enabled` field is `true` in the shoot-rsyslog-relp extension configuration, a value for `.tls.secretReferenceName` must also be specified so that it references a [named resource reference](https://github.com/gardener/gardener/blob/v1.82.0/pkg/apis/core/v1beta1/types_shoot.go#L487) in the shoot's `.spec.resources[]` array. 
    The webhook appends the data of the referenced secret in the Shoot's control plane namespace to the OperatingSystemConfig files.
    1. The webhook appends the `rsyslog-configurator.service` unit to the OperatingSystemConfig units. The unit invokes the `configure-rsyslog.sh` script every 15 seconds.

#### Extension disablement

This section outlines how the extension disablement works, i.e the extension has be removed from the Shoot spec.

1. As part of the Shoot reconciliation flow, gardenlet destroys the [Extension](https://github.com/gardener/gardener/blob/v1.82.0/docs/extensions/extension.md) resource because it is no longer needed.
   1. As part of the deletion flow, the shoot-rsyslog-relp extension deploys the [`rsyslog-relp-configuration-cleaner` DaemonSet](../../pkg/component/rsyslogrelpconfigcleaner/rsyslog_relp_config_cleaner.go) to the Shoot cluster to clean up the existing rsyslog configuration and revert the audit rules.

## Shoot deletion

This section outlines how the deletion works for a Shoot with the shoot-rsyslog-relp extension enabled.

1. As part of the Shoot deletion flow, gardenlet destroys the [Extension](https://github.com/gardener/gardener/blob/v1.82.0/docs/extensions/extension.md) resource.
   1. In the Shoot deletion flow the Extension resource is deleted after the Worker resource. Hence, there is no need to deploy the  [`rsyslog-relp-configuration-cleaner` DaemonSet](../../pkg/component/rsyslogrelpconfigcleaner/rsyslog_relp_config_cleaner.go) to the Shoot cluster to clean up the existing rsyslog configuration and revert the audit rules.