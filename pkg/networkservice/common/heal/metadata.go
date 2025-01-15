// Copyright (c) 2021-2023 Cisco and/or its affiliates.
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

package heal

import (
	"context"

	"github.com/ljkiraly/sdk/pkg/networkservice/utils/metadata"
)

type key struct{}

type eventLoopHandle struct {
	cancel           context.CancelFunc
	healingStartedCh <-chan bool
	healingStarted   bool
}

// store sets the eventLoopHandle stored in per Connection.Id metadata.
func store(ctx context.Context, cancel eventLoopHandle) {
	metadata.Map(ctx, true).Store(key{}, cancel)
}

// loadAndDelete deletes the eventLoopHandle stored in per Connection.Id metadata,
// returning the previous value if any. The loaded result reports whether the key was present.
func loadAndDelete(ctx context.Context) (value eventLoopHandle, loaded bool) {
	rawValue, ok := metadata.Map(ctx, true).LoadAndDelete(key{})
	if !ok {
		return
	}
	value, ok = rawValue.(eventLoopHandle)
	return value, ok
}
