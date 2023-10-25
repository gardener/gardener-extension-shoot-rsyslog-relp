// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package rsyslogrlep

import (
	"errors"
	"os"

	"github.com/gardener/gardener/extensions/pkg/controller/cmd"
	extensionsheartbeatcontroller "github.com/gardener/gardener/extensions/pkg/controller/heartbeat"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	apisconfig "github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/config"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/config/v1alpha1"
	controllerconfig "github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/controller/config"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/controller/lifecycle"
)

var (
	scheme  *runtime.Scheme
	decoder runtime.Decoder
)

func init() {
	scheme = runtime.NewScheme()
	utilruntime.Must(apisconfig.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))

	decoder = serializer.NewCodecFactory(scheme).UniversalDecoder()
}

// Options holds options related to the rsyslog relp extension.
type Options struct {
	ConfigLocation string
	config         *RsyslogRelpServiceConfig
}

// AddFlags implements Flagger.AddFlags.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.ConfigLocation, "config", "", "Path to the rsyslog relp extension configuration")
}

// Complete implements Completer.Complete.
func (o *Options) Complete() error {
	if o.ConfigLocation == "" {
		return errors.New("config location is not set")
	}

	data, err := os.ReadFile(o.ConfigLocation)
	if err != nil {
		return err
	}

	configuration := apisconfig.Configuration{}
	_, _, err = decoder.Decode(data, nil, &configuration)
	if err != nil {
		return err
	}

	o.config = &RsyslogRelpServiceConfig{
		config: configuration,
	}

	return nil
}

// Completed returns the decoded Configuration instance. Only call this if `Complete` was successful.
func (o *Options) Completed() *RsyslogRelpServiceConfig {
	return o.config
}

// RsyslogRelpServiceConfig contains configuration information about the rsyslog relp service.
type RsyslogRelpServiceConfig struct {
	config apisconfig.Configuration
}

// Apply applies the Options to the passed ControllerOptions instance.
func (c *RsyslogRelpServiceConfig) Apply(config *controllerconfig.Config) {
	config.Configuration = c.config
}

// ControllerSwitches are the cmd.ControllerSwitches for the extension controllers.
func ControllerSwitches() *cmd.SwitchOptions {
	return cmd.NewSwitchOptions(
		cmd.Switch(lifecycle.Name, lifecycle.AddToManager),
		cmd.Switch(extensionsheartbeatcontroller.ControllerName, extensionsheartbeatcontroller.AddToManager),
	)
}
