// Copyright (c) 2022 Cisco and/or its affiliates.
//
// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
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

package swapip_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ghodss/yaml"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/ljkiraly/sdk/pkg/networkservice/common/swapip"
	"github.com/ljkiraly/sdk/pkg/networkservice/core/next"
	"github.com/ljkiraly/sdk/pkg/networkservice/utils/checks/checkrequest"
	"github.com/ljkiraly/sdk/pkg/networkservice/utils/checks/checkresponse"
	"github.com/ljkiraly/sdk/pkg/tools/fs"
)

func TestSwapIPServer_Request(t *testing.T) {
	defer goleak.VerifyNone(t)

	p1 := filepath.Join(t.TempDir(), "map-ip-1.yaml")
	p2 := filepath.Join(t.TempDir(), "map-ip-2.yaml")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := os.WriteFile(p1, []byte(`172.16.2.10: 172.16.1.10`), os.ModePerm)
	require.NoError(t, err)

	err = os.WriteFile(p2, []byte(`172.16.2.100: 172.16.1.100`), os.ModePerm)
	require.NoError(t, err)

	ch1 := convertBytesChToMapCh(fs.WatchFile(ctx, p1))
	ch2 := convertBytesChToMapCh(fs.WatchFile(ctx, p2))

	var testChain = next.NewNetworkServiceServer(
		/* Source side */
		checkresponse.NewServer(t, func(t *testing.T, c *networkservice.Connection) {
			require.Equal(t, "172.16.1.100", c.Mechanism.Parameters[common.DstIP])
			require.Equal(t, "172.16.2.100", c.Mechanism.Parameters[common.DstOriginalIP])
			c.Mechanism.Parameters[common.SrcOriginalIP] = ""
		}),
		swapip.NewServer(ch1),
		checkrequest.NewServer(t, func(t *testing.T, r *networkservice.NetworkServiceRequest) {
			require.Equal(t, "172.16.2.10", r.Connection.Mechanism.Parameters[common.SrcIP])
			require.Equal(t, "", r.Connection.Mechanism.Parameters[common.SrcOriginalIP])
			r.Connection.Mechanism.Parameters[common.SrcOriginalIP] = "172.16.2.10"
		}),
		/* Destination side */
		checkresponse.NewServer(t, func(t *testing.T, c *networkservice.Connection) {
			require.Equal(t, "172.16.1.100", c.Mechanism.Parameters[common.DstIP])
			require.Equal(t, "172.16.2.100", c.Mechanism.Parameters[common.DstOriginalIP])
		}),
		swapip.NewServer(ch2),
		checkrequest.NewServer(t, func(t *testing.T, r *networkservice.NetworkServiceRequest) {
			require.Equal(t, "", r.Connection.Mechanism.Parameters[common.DstOriginalIP])
			require.Equal(t, "", r.Connection.Mechanism.Parameters[common.DstIP])
			r.Connection.Mechanism.Parameters[common.DstIP] = "172.16.2.100"
		}),
	)

	r := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Mechanism: &networkservice.Mechanism{
				Parameters: map[string]string{
					common.SrcIP: "172.16.2.10",
				},
			},
		},
	}

	time.Sleep(time.Second / 4)

	resp, err := testChain.Request(context.Background(), r)
	require.NoError(t, err)

	// refresh
	_, err = testChain.Request(ctx, &networkservice.NetworkServiceRequest{Connection: resp})
	require.NoError(t, err)
}

func convertBytesChToMapCh(in <-chan []byte) <-chan map[string]string {
	var out = make(chan map[string]string)
	go func() {
		for data := range in {
			var r map[string]string
			_ = yaml.Unmarshal(data, &r)
			out <- r
		}
		close(out)
	}()

	return out
}
