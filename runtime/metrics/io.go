// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"fmt"

	"golang.org/x/exp/maps"
	"github.com/ServiceWeaver/weaver/runtime/protos"
)

// An Exporter produces MetricUpdates summarizing the change in metrics over
// time.
type Exporter struct {
	versions map[uint64]uint64
}

// Export produces a MetricUpdate that summarizes the changes to all metrics
// since the last call to MetricUpdate.
func (e *Exporter) Export() *protos.MetricUpdate {
	if e.versions == nil {
		e.versions = map[uint64]uint64{}
	}

	metricsMu.RLock()
	defer metricsMu.RUnlock()

	update := &protos.MetricUpdate{}
	for _, metric := range metrics {
		metric.Init()
		latest, ok := e.versions[metric.id]
		if !ok {
			e.versions[metric.id] = metric.Version()
			update.Defs = append(update.Defs, metric.MetricDef())
			update.Values = append(update.Values, metric.MetricValue())
			continue
		}

		version := metric.Version()
		if version == latest {
			continue
		}
		e.versions[metric.id] = version
		update.Values = append(update.Values, metric.MetricValue())
	}
	return update
}

// An Importer maintains a snapshot of all metric values, updating over time
// using the MetricUpdates generated by an Exporter.
type Importer struct {
	metrics map[uint64]*MetricSnapshot
}

// Import updates the Importer's snapshot with the latest metric changes.
func (i *Importer) Import(update *protos.MetricUpdate) ([]*MetricSnapshot, error) {
	if i.metrics == nil {
		i.metrics = map[uint64]*MetricSnapshot{}
	}

	for _, def := range update.Defs {
		if _, ok := i.metrics[def.Id]; ok {
			return nil, fmt.Errorf("metrics.Importer: duplicate MetricDef %d", def.Id)
		}
		i.metrics[def.Id] = &MetricSnapshot{
			Id:     def.Id,
			Name:   def.Name,
			Type:   def.Typ,
			Help:   def.Help,
			Labels: def.Labels,
			Bounds: def.Bounds,
		}
	}

	for _, val := range update.Values {
		metric, ok := i.metrics[val.Id]
		if !ok {
			return nil, fmt.Errorf("metrics.Importer: unknown metric %d", val.Id)
		}
		metric.Value = val.Value
		metric.Counts = val.Counts
	}

	return maps.Values(i.metrics), nil
}
