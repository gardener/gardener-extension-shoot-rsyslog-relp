# Deploying Rsyslog Relp Extension Locally

This document will walk you through running the Rsyslog Relp extension and a fake rsyslog relp service on your local machine for development purposes. This guide uses the Gardener's local development setup and builds on top of it.
If you encounter difficulties, please open an issue so that we can make this process easier.

## Prerequisites

- Make sure that you have a running local Gardener setup. The steps to complete this can be found [here](https://github.com/gardener/gardener/blob/master/docs/deployment/getting_started_locally.md).
- Make sure you are running Gardener version >= 1.74.0 or the latest version of the master branch.

## Setting up the Rsyslog Relp Extension

**Important:** Make sure that your `KUBECONFIG` env variable is targeting the local Gardener cluster!

```bash
make extension-up
```

This will build the `shoot-rsyslog-relp`, `shoot-rsyslog-relp-admission` and `shoot-rsyslog-relp-echo-server` images and deploy the needed resources and configurations in the Garden cluster. The `shoot-rsyslog-relp-echo-server` will act as development replacement of a real rsyslog relp server.

## Creating a `Shoot` cluster

Once the above step is completed we can deploy and configure a shoot cluster with a default rsyslog relp settings.

```bash
kubectl apply -f ./example/shoot.yaml
```

Once the shoot's namespace is created we can create a `networkpolicy` that will allow egress traffic from the `rsyslog` on the `Shoot`'s nodes to the `rsyslog-relp-echo-server` that serves as a fake rsyslog target server.

```bash
kubectl apply -f ./example/local/allow-machine-to-rsyslog-relp-echo-server-netpol.yaml
```

Currently the shoot's nodes run Ubuntu which does not have the `rsyslog-relp` and `auditd` packages installed, so the configuration done by the extension has no effect.
Once the shoot is created, we have to manually install the `rsyslog-relp` and `auditd` packages:

```bash
kubectl -n shoot--local--local exec -it $(kubectl -n shoot--local--local get po -l app=machine,machine-provider=local -o name) -- bash -c "
   apt-get update && \
   apt-get install -y rsyslog-relp auditd && \
   systemctl enable rsyslog.service && \
   systemctl start rsyslog.service"
```

After that is done, we can verify that log messages are forwarded to the `rsyslog-relp-echo-server` by checking its logs.

```bash
kubectl -n rsyslog-relp-echo-server logs deployment/rsyslog-relp-echo-server
```

## Making changes to the Rsyslog Relp Extension

Changes to the rsyslog relp extension can be applied to the local environment by repeatedly running the `make` recipe.

```bash
make extension-up
```

## Tearing down the dev environment

To tear down the development environment delete the `Shoot` cluster or disable the `shoot-rsyslog-relp` extension in th `Shoot`'s spec. When the extension is not used by the `Shoot` anymore, you can run:

```bash
make extension-down
```

This will delete the `ControllerRegistration` and `ControllerDeployment` of the extension, the `shoot-rsyslog-relp-admission` deployment and the `rsyslog-relp-echo-server` deployment.

# Maintaining the publicly available image for the rsyslog-relp echo server

The [testmachinery tests](../../test/testmachinery/shoot/) use an `rsyslog-relp-echo-server` image from a publicly available repository. The one which is currently used is `eu.gcr.io/gardener-project/gardener/extensions/shoot-rsyslog-relp-echo-server:v0.1.0`.

Sometimes it might be necessary to update the image and publish it, e.g. when updating the `alpine` base image version specified in the repository's [Dokerfile](../../Dockerfile#L34).

To do that:
1. Bump the version with which the image is built in the [Makefile](../../Makefile#L14).
1. Build the `shoot-rsyslog-relp-echo-server` image:
   ```bash
   make echo-server-docker-image
   ```

1. Once the image is built, push it to `gcr` with:
   ```bash
   make push-echo-server-image
   ```

1. Finally, bump the version of the image used by the `testmachinery` tests [here](../../test/testmachinery/shoot/common_test.go).
1. Create a PR with the changes.