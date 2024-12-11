// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package om7scanextension // import "github.com/insoft-icnp/OM7-Collector/extension/om7scanextension"

import (
	"context"
	"errors"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"go.uber.org/zap"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"sync/atomic"
	"time"
)

// SummaryPipelinesTableData contains data for pipelines summary table template.
type SummaryPipelinesTableData struct {
	Pipeline []SummaryPipelinesTableRowData
}

// SummaryPipelinesTableRowData contains data for one row in pipelines summary table template.
type SummaryPipelinesTableRowData struct {
	FullName    string
	InputType   string
	MutatesData bool
	Receivers   []string
	Processors  []string
	Exporters   []string
}

var running = &atomic.Bool{}

type om7scanExtension struct {
	config       *Config
	telemetry    component.TelemetrySettings
	cancel       context.CancelFunc
	clientConn   *grpc.ClientConn
	logger       *zap.Logger
	traceClient  ptraceotlp.GRPCClient
	metricClient pmetricotlp.GRPCClient
	metadata     metadata.MD
	callOptions  []grpc.CallOption
}

func newExtension(config *Config, telemetry component.TelemetrySettings) *om7scanExtension {
	return &om7scanExtension{
		config:    config,
		telemetry: telemetry,
	}
}

func (om *om7scanExtension) Start(ctx context.Context, host component.Host) error {
	if !running.CompareAndSwap(false, true) {
		return errors.New("only a single om7scan extension instance can be running per process")
	}

	var startErr error
	defer func() {
		if startErr != nil {
			running.Store(false)
		}
	}()

	piplineData, ok := host.(interface {
		GetOm7Scan() *[]byte
		GetFactory(kind component.Kind, componentType component.Type) component.Factory
	})

	rawData := piplineData.GetOm7Scan()

	ctx, om.cancel = context.WithCancel(ctx)

	om.telemetry.Logger.Info("Register to Om7ScanExtension", zap.Any("Interval Seconds : ", om.config.CollectionInterval), zap.Any("Bool", ok))

	if om.config.CollectionInterval != 0 {
		tickter := time.NewTicker(om.config.CollectionInterval)
		go func() {
			for {
				select {
				case <-tickter.C:

					om.telemetry.Logger.Info("테스트시작")
					traceData, err := ConvertPipelineToTrace(rawData)
					if err != nil {
						om.telemetry.Logger.Error("Failed to Converted that Trace Data", zap.Error(err))
						tickter.Stop()
						return
					}

					if om.clientConn, err = om.config.ToClientConn(ctx, host, om.telemetry); err != nil {
						tickter.Stop()
						return
					}

					om.traceClient = ptraceotlp.NewGRPCClient(om.clientConn)
					om.metricClient = pmetricotlp.NewGRPCClient(om.clientConn)

					headers := map[string]string{}
					for k, v := range om.config.ClientConfig.Headers {
						headers[k] = string(v)
					}

					om.metadata = metadata.New(headers)
					om.callOptions = []grpc.CallOption{
						grpc.WaitForReady(om.config.ClientConfig.WaitForReady),
					}

					if err = om.pushTraces(ctx, traceData); err != nil {
						om.telemetry.Logger.Error("Failed to PushTrace", zap.Error(err))
						tickter.Stop()
						return
					}

					om.telemetry.Logger.Info("테스트시끝")
				case <-ctx.Done():
					tickter.Stop()
					return
				}
			}
		}()
	}

	return nil
}

func (om *om7scanExtension) Shutdown(context.Context) error {
	if om.cancel != nil {
		om.cancel()
	}
	return errors.New("Shutdown")
}

func (om *om7scanExtension) enhanceContext(ctx context.Context) context.Context {
	if om.metadata.Len() > 0 {
		return metadata.NewOutgoingContext(ctx, om.metadata)
	}
	return ctx
}

func (om *om7scanExtension) pushTraces(ctx context.Context, traceData ptrace.Traces) error {
	req := ptraceotlp.NewExportRequestFromTraces(traceData)
	resp, respErr := om.traceClient.Export(om.enhanceContext(ctx), req, om.callOptions...)
	if err := processError(respErr); err != nil {
		om.telemetry.Logger.Error("Failed to traceClient Data", zap.Error(err))
		return err
	}
	partialSuccess := resp.PartialSuccess()
	if !(partialSuccess.ErrorMessage() == "" && partialSuccess.RejectedSpans() == 0) {
		om.telemetry.Logger.Warn("Partial success response",
			zap.String("message", resp.PartialSuccess().ErrorMessage()),
			zap.Int64("dropped_data_points", resp.PartialSuccess().RejectedSpans()),
		)
	}

	return nil
}

func (om *om7scanExtension) pushMetrics(ctx context.Context, md pmetric.Metrics) error {
	req := pmetricotlp.NewExportRequestFromMetrics(md)
	resp, respErr := om.metricClient.Export(om.enhanceContext(ctx), req, om.callOptions...)
	if err := processError(respErr); err != nil {
		return err
	}
	partialSuccess := resp.PartialSuccess()
	if !(partialSuccess.ErrorMessage() == "" && partialSuccess.RejectedDataPoints() == 0) {
		om.telemetry.Logger.Warn("Partial success response",
			zap.String("message", resp.PartialSuccess().ErrorMessage()),
			zap.Int64("dropped_data_points", resp.PartialSuccess().RejectedDataPoints()),
		)
	}
	return nil
}

func shouldRetry(code codes.Code, retryInfo *errdetails.RetryInfo) bool {
	switch code {
	case codes.Canceled,
		codes.DeadlineExceeded,
		codes.Aborted,
		codes.OutOfRange,
		codes.Unavailable,
		codes.DataLoss:
		// These are retryable errors.
		return true
	case codes.ResourceExhausted:
		// Retry only if RetryInfo was supplied by the server.
		// This indicates that the server can still recover from resource exhaustion.
		return retryInfo != nil
	}
	// Don't retry on any other code.
	return false
}

func processError(err error) error {
	if err == nil {
		// Request is successful, we are done.
		return nil
	}

	// We have an error, check gRPC status code.
	st := status.Convert(err)
	if st.Code() == codes.OK {
		// Not really an error, still success.
		return nil
	}

	// Now, this is a real error.
	retryInfo := getRetryInfo(st)

	if !shouldRetry(st.Code(), retryInfo) {
		// It is not a retryable error, we should not retry.
		return consumererror.NewPermanent(err)
	}

	// Check if server returned throttling information.
	throttleDuration := getThrottleDuration(retryInfo)
	if throttleDuration != 0 {
		// We are throttled. Wait before retrying as requested by the server.
		return exporterhelper.NewThrottleRetry(err, throttleDuration)
	}

	// Need to retry.
	return err
}

func getRetryInfo(status *status.Status) *errdetails.RetryInfo {
	for _, detail := range status.Details() {
		if t, ok := detail.(*errdetails.RetryInfo); ok {
			return t
		}
	}
	return nil
}

func getThrottleDuration(t *errdetails.RetryInfo) time.Duration {
	if t == nil || t.RetryDelay == nil {
		return 0
	}
	if t.RetryDelay.Seconds > 0 || t.RetryDelay.Nanos > 0 {
		return time.Duration(t.RetryDelay.Seconds)*time.Second + time.Duration(t.RetryDelay.Nanos)*time.Nanosecond
	}
	return 0
}
