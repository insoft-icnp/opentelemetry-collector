// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph // import "github.com/insoft-icnp/OM7-Collector/service/internal/graph"

import (
	"encoding/json"
	"sort"
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

func (g *Graph) ScanPipeline() *[]byte {
	sumData := SummaryPipelinesTableData{}
	sumData.Pipeline = make([]SummaryPipelinesTableRowData, 0, len(g.pipelines))
	for c, p := range g.pipelines {
		recvIDs := make([]string, 0, len(p.receivers))
		for _, c := range p.receivers {
			switch n := c.(type) {
			case *receiverNode:
				recvIDs = append(recvIDs, n.componentID.String())
			case *connectorNode:
				recvIDs = append(recvIDs, n.componentID.String()+" (connector)")
			}
		}
		procIDs := make([]string, 0, len(p.processors))
		for _, c := range p.processors {
			procIDs = append(procIDs, c.componentID.String())
		}
		exprIDs := make([]string, 0, len(p.exporters))
		for _, c := range p.exporters {
			switch n := c.(type) {
			case *exporterNode:
				exprIDs = append(exprIDs, n.componentID.String())
			case *connectorNode:
				exprIDs = append(exprIDs, n.componentID.String()+" (connector)")
			}
		}

		sumData.Pipeline = append(sumData.Pipeline, SummaryPipelinesTableRowData{
			FullName:    c.String(),
			InputType:   c.Signal().String(),
			MutatesData: p.capabilitiesNode.getConsumer().Capabilities().MutatesData,
			Receivers:   recvIDs,
			Processors:  procIDs,
			Exporters:   exprIDs,
		})
	}
	sort.Slice(sumData.Pipeline, func(i, j int) bool {
		return sumData.Pipeline[i].FullName < sumData.Pipeline[j].FullName
	})

	data, _ := json.Marshal(sumData)

	return &data
}
