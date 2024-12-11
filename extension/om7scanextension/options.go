package om7scanextension

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"time"
)

type PipelineData struct {
	Pipeline []Pipeline `json:"Pipeline"`
}

type Pipeline struct {
	FullName    string   `json:"FullName"`
	InputType   string   `json:"InputType"`
	MutatesData bool     `json:"MutatesData"`
	Receivers   []string `json:"Receivers"`
	Processors  []string `json:"Processors"`
	Exporters   []string `json:"Exporters"`
}

func ConvertPipelineToTrace(jsonData *[]byte) (ptrace.Traces, error) {
	var pipelineData PipelineData

	if err := json.Unmarshal(*jsonData, &pipelineData); err != nil {
		return ptrace.Traces{}, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	traces := ptrace.NewTraces()
	rs := traces.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("service.name", "collector-pipelines")
	ils := rs.ScopeSpans().AppendEmpty()
	spans := ils.Spans()

	traceId := newTraceID()

	var previousSpanID pcommon.SpanID

	for idx, pipeline := range pipelineData.Pipeline {
		span := spans.AppendEmpty()

		span.SetName(pipeline.FullName)

		span.SetTraceID(traceId)
		spenID := newSegmentID()
		span.SetSpanID(spenID)

		if idx > 0 {
			span.SetParentSpanID(previousSpanID)
		}

		previousSpanID = spenID

		span.Attributes().PutStr("input_type", pipeline.InputType)
		span.Attributes().PutBool("mutates_data", pipeline.MutatesData)
		span.Attributes().PutStr("receivers", fmt.Sprintf("%v", pipeline.Receivers))
		span.Attributes().PutStr("processors", fmt.Sprintf("%v", pipeline.Processors))
		span.Attributes().PutStr("exporters", fmt.Sprintf("%v", pipeline.Exporters))

		// Simulate timestamps for start and end
		span.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		span.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	}

	return traces, nil
}

//func ConvertPipelineToMetric(jsonData *[]byte) (pmetric.Metrics, error) {
//	var pipelineData PipelineData
//
//	if err := json.Unmarshal(*jsonData, &pipelineData); err != nil {
//		return pmetric.Metrics{}, err
//	}
//	metrics := pmetric.NewMetrics()
//	ilm := metrics.ResourceMetrics().AppendEmpty()
//
//	for _, pipeline := range pipelineData.Pipeline {
//		ilm.Resource().Attributes().PutStr("input_type", pipeline.InputType)
//		ilm.Resource().Attributes().PutBool("mutates_data", pipeline.MutatesData)
//		ilm.Resource().Attributes().PutStr("receivers", fmt.Sprintf("%v", pipeline.Receivers))
//		ilm.Resource().Attributes().PutStr("processors", fmt.Sprintf("%v", pipeline.Processors))
//		ilm.Resource().Attributes().PutStr("exporters", fmt.Sprintf("%v", pipeline.Exporters))
//	}
//
//	s1 := ilm.ScopeMetrics().AppendEmpty()
//	m := s1.Metrics().AppendEmpty()
//	m.SetName("metricTest")
//	dp := m.SetEmptyGauge().DataPoints().AppendEmpty()
//	dp.SetIntValue(1)
//
//	return dp, nil
//}

func newTraceID() pcommon.TraceID {
	var traceID [16]byte
	_, _ = rand.Read(traceID[:]) // 랜덤 값으로 채움
	return pcommon.TraceID(traceID)
}

func newSegmentID() pcommon.SpanID {
	var spanID [8]byte          // SpanID는 보통 8바이트로 구성됩니다.
	_, _ = rand.Read(spanID[:]) // 랜덤 값으로 채우기
	return pcommon.SpanID(spanID)
}
