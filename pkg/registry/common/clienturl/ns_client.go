// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
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

package clienturl

import (
	"context"
	"net/url"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	"github.com/ljkiraly/sdk/pkg/registry/core/next"
	"github.com/ljkiraly/sdk/pkg/tools/clienturlctx"
)

type clientURLNSClient struct {
	u *url.URL
}

func (c *clientURLNSClient) Register(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	ctx = clienturlctx.WithClientURL(ctx, c.u)
	return next.NetworkServiceRegistryClient(ctx).Register(ctx, in, opts...)
}

func (c *clientURLNSClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	ctx = clienturlctx.WithClientURL(ctx, c.u)
	return next.NetworkServiceRegistryClient(ctx).Find(ctx, in, opts...)
}

func (c *clientURLNSClient) Unregister(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	ctx = clienturlctx.WithClientURL(ctx, c.u)
	return next.NetworkServiceRegistryClient(ctx).Unregister(ctx, in, opts...)
}

// NewNetworkServiceRegistryClient - returns a new null client that does nothing but call next.NetworkServiceRegistryClient(ctx).
func NewNetworkServiceRegistryClient(u *url.URL) registry.NetworkServiceRegistryClient {
	return &clientURLNSClient{
		u: u,
	}
}
