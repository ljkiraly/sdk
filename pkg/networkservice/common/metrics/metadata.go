// Copyright (c) 2023 Nordix Foundation.
//
// Copyright (c) 2022-2024 Cisco and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/metric"

	"github.com/ljkiraly/sdk/pkg/networkservice/utils/metadata"
)

type keyType struct{}

type metricsData struct {
	counter  map[string]metric.Int64Counter
	previous sync.Map
}

func loadOrStore(ctx context.Context, metrics *metricsData) (value *metricsData, ok bool) {
	rawValue, ok := metadata.Map(ctx, false).LoadOrStore(keyType{}, metrics)
	return rawValue.(*metricsData), ok
}
