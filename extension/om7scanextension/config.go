// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package om7scanextension // import "github.com/insoft-icnp/OM7-Collector/extension/om7scanextension"

import (
	"errors"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

// Config has the configuration for the extension enabling the om7scan extension.
type Config struct {
	scraperhelper.ControllerConfig `mapstructure:",squash"`
	configgrpc.ClientConfig        `mapstructure:",squash"`
	// otlpExporter
}

var _ component.Config = (*Config)(nil)

// Validate checks if the extension configuration is valid
func (cfg *Config) Validate() error {
	if cfg.CollectionInterval == 0 && cfg.ClientConfig.Endpoint == "" {
		return errors.New("\"collection-interval, Endpoint\" is required when using the \"om7scan\" extension")
	}
	return nil
}
