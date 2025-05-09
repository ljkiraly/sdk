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

// Package checkclose - provides networkservice chain elements to check the Close received from the previous element in the chain
package checkclose

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/ljkiraly/sdk/pkg/networkservice/core/next"
)

type checkCloseClient struct {
	*testing.T
	check func(*testing.T, *networkservice.Connection)
}

// NewClient - returns NetworkServiceClient chain elements to check the Close received from the previous element in the chain
//
//	t - *testing.T for checks
//	check - function to check the Connection
func NewClient(t *testing.T, check func(*testing.T, *networkservice.Connection)) networkservice.NetworkServiceClient {
	return &checkCloseClient{
		T:     t,
		check: check,
	}
}

func (c *checkCloseClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (c *checkCloseClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	c.check(c.T, conn)
	return next.Client(ctx).Close(ctx, conn, opts...)
}
