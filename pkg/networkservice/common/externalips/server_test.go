// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2022 Cisco Systems, Inc.
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

package externalips_test

import (
	"context"
	"net"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/ljkiraly/sdk/pkg/networkservice/common/externalips"
	"github.com/ljkiraly/sdk/pkg/networkservice/core/next"
	"github.com/ljkiraly/sdk/pkg/networkservice/utils/checks/checkcontext"
)

func TestExternalIPsServer_SourceModifying(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	timeout := time.After(time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	updateCh := make(chan map[string]string)
	server := externalips.NewServer(ctx, externalips.WithUpdateChannel(updateCh))

	sendUpdate := func(update map[string]string) {
		select {
		case <-timeout:
			t.Fatal("timeout while sending update")
		case updateCh <- update:
			return
		}
	}
	checkUpdate := func(internal, external string) {
		done := false
		for !done {
			select {
			case <-timeout:
				t.Fatal("timeout during wait changes from update")
			default:
			}
			_, _ = next.NewNetworkServiceServer(server, checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
				if externalips.FromInternal(ctx, net.ParseIP(internal)).Equal(net.ParseIP(external)) {
					done = true
				}
			})).Request(ctx, &networkservice.NetworkServiceRequest{})
		}
	}

	sendUpdate(map[string]string{"172.16.1.1": "180.17.2.1"})
	checkUpdate("172.16.1.1", "180.17.2.1")

	sendUpdate(map[string]string{"172.16.1.1": "180.17.2.1", "127.0.0.1": "180.17.2.2"})
	checkUpdate("127.0.0.1", "180.17.2.2")

	sendUpdate(map[string]string{})
	checkUpdate("127.0.0.1", "")
}

func TestExternalIPsServer_NoFile(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tmpPath := path.Join(os.TempDir(), t.Name())
	_ = os.MkdirAll(tmpPath, os.ModePerm)
	defer func() {
		_ = os.RemoveAll(tmpPath)
	}()
	internalIP := net.ParseIP("127.0.0.1")
	externalIP := net.ParseIP("180.20.1.1")
	filePath := filepath.Join(tmpPath, "file.txt")
	server := externalips.NewServer(ctx, externalips.WithFilePath(filePath))
	checkChain := next.NewNetworkServiceServer(server, checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
		require.Nil(t, externalips.FromInternal(ctx, internalIP))
		require.Nil(t, externalips.ToInternal(ctx, externalIP))
	}))
	_, err := checkChain.Request(context.Background(), &networkservice.NetworkServiceRequest{})
	require.NoError(t, err)
	err = os.WriteFile(filePath, []byte(internalIP.String()+": "+externalIP.String()), os.ModePerm)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		var result bool
		checkChain = next.NewNetworkServiceServer(server, checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
			actualExternal := externalips.FromInternal(ctx, internalIP)
			if actualExternal == nil {
				return
			}
			actualInternal := externalips.ToInternal(ctx, externalIP)
			if actualInternal == nil {
				return
			}
			result = actualExternal.Equal(externalIP) &&
				actualInternal.Equal(internalIP)
		}))
		_, err = checkChain.Request(context.Background(), &networkservice.NetworkServiceRequest{})
		require.NoError(t, err)
		return result
	}, time.Second, time.Millisecond*100)
}
