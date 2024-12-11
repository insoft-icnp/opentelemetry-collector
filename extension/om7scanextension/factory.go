// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package om7scanextension // import "github.com/insoft-icnp/OM7-Collector/extension/om7scanextension"

import (
	"context"
	"go.opentelemetry.io/collector/config/configcompression"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/receiver/scraperhelper"

	"github.com/insoft-icnp/opentelemetry-collector/extension/om7scanextension/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

const (
	defaultEndpoint = "localhost:17818"
)

// NewFactory creates a factory for Z-Pages extension.
func NewFactory() extension.Factory {
	return extension.NewFactory(
		metadata.Type,
		createDefaultConfig,
		create,
		metadata.ExtensionStability)
}

func createDefaultConfig() component.Config {
	return &Config{
		ControllerConfig: scraperhelper.NewDefaultControllerConfig(),
		ClientConfig: configgrpc.ClientConfig{
			Headers: map[string]configopaque.String{},
			// Default to gzip compression
			Compression: configcompression.TypeGzip,
			// We almost read 0 bytes, so no need to tune ReadBufferSize.
			WriteBufferSize: 512 * 1024,
		},
	}
}

// create creates the extension based on this config.
func create(_ context.Context, set extension.Settings, cfg component.Config) (extension.Extension, error) {
	return newExtension(cfg.(*Config), set.TelemetrySettings), nil
}
