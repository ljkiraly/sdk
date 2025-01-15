// Copyright (c) 2022 Cisco and/or its affiliates.
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

package begin_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"

	"github.com/ljkiraly/sdk/pkg/registry/common/begin"
	"github.com/ljkiraly/sdk/pkg/registry/core/adapters"
	"github.com/ljkiraly/sdk/pkg/registry/core/chain"
)

func TestSerializeBoth_StressTest(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := chain.NewNetworkServiceEndpointRegistryServer(
		begin.NewNetworkServiceEndpointRegistryServer(),
		newParallelServer(t),
		adapters.NetworkServiceEndpointClientToServer(chain.NewNetworkServiceEndpointRegistryClient(
			begin.NewNetworkServiceEndpointRegistryClient(),
			newParallelClient(t),
		),
		),
	)

	wg := new(sync.WaitGroup)
	wg.Add(parallelCount)
	for i := 0; i < parallelCount; i++ {
		go func(id string) {
			defer wg.Done()

			resp, err := server.Register(ctx, &registry.NetworkServiceEndpoint{
				Name: id,
			})
			assert.NoError(t, err)

			_, err = server.Unregister(ctx, resp)
			assert.NoError(t, err)
		}(fmt.Sprint(i % 20))
	}
	wg.Wait()
}
