// Copyright (c) 2020 Cisco Systems, Inc.
//
// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

// Package chain provides a simple file for creating a networkservice.NetworkService{Server,Client} from a 'chain' of
// networkservice.NetworkService{Server,Client}s
package chain

import (
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/ljkiraly/sdk/pkg/networkservice/core/next"
	"github.com/ljkiraly/sdk/pkg/networkservice/core/trace"
)

// NewNetworkServiceServer - chains together a list of networkservice.Servers with tracing
func NewNetworkServiceServer(servers ...networkservice.NetworkServiceServer) networkservice.NetworkServiceServer {
	return next.NewNetworkServiceServer(
		next.NewWrappedNetworkServiceServer(trace.NewNetworkServiceServer, servers...),
	)
}
