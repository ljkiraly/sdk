// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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

package dnsresolve

import (
	"context"
	"net"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/ljkiraly/sdk/pkg/tools/clienturlctx"
	"github.com/ljkiraly/sdk/pkg/tools/interdomain"

	"github.com/ljkiraly/sdk/pkg/registry/core/next"
)

type dnsNSEResolveClient struct {
	resolver        Resolver
	registryService string
}

// NewNetworkServiceEndpointRegistryClient creates new NetworkServiceEndpointRegistryClient that can resolve passed domain to clienturl
func NewNetworkServiceEndpointRegistryClient(opts ...Option) registry.NetworkServiceEndpointRegistryClient {
	var clientOptions = &options{
		resolver:        net.DefaultResolver,
		registryService: DefaultRegistryService,
	}

	for _, opt := range opts {
		opt(clientOptions)
	}

	r := &dnsNSEResolveClient{
		resolver:        clientOptions.resolver,
		registryService: clientOptions.registryService,
	}

	return r
}

func (d *dnsNSEResolveClient) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	var domain = resolveNSE(nse)
	var u, err = resolveDomain(ctx, d.registryService, domain, d.resolver)

	if err != nil {
		return nil, err
	}

	ctx = clienturlctx.WithClientURL(ctx, u)

	translateNSE(nse, interdomain.Target)

	resp, err := next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, nse, opts...)

	if err != nil {
		return nil, err
	}

	translateNSE(resp, func(s string) string {
		return interdomain.Join(s, domain)
	})

	return resp, nil
}

type dnsNSEResolveFindClient struct {
	registry.NetworkServiceEndpointRegistry_FindClient
	domain string
}

func (c *dnsNSEResolveFindClient) Recv() (*registry.NetworkServiceEndpointResponse, error) {
	resp, err := c.NetworkServiceEndpointRegistry_FindClient.Recv()
	if err != nil {
		return resp, err
	}

	translateNSE(resp.NetworkServiceEndpoint, func(str string) string {
		return interdomain.Join(str, c.domain)
	})

	return resp, nil
}

func (d *dnsNSEResolveClient) Find(ctx context.Context, q *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	var domain = resolveNSE(q.NetworkServiceEndpoint)
	var nsmgrProxyURL, err = resolveDomain(ctx, d.registryService, domain, d.resolver)

	if err != nil {
		return nil, err
	}

	ctx = clienturlctx.WithClientURL(ctx, nsmgrProxyURL)
	translateNSE(q.NetworkServiceEndpoint, interdomain.Target)

	resp, err := next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, q, opts...)
	if err != nil {
		return nil, err
	}

	return &dnsNSEResolveFindClient{
		NetworkServiceEndpointRegistry_FindClient: resp,
		domain: domain,
	}, nil
}

func (d *dnsNSEResolveClient) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	var domain = resolveNSE(nse)
	var u, err = resolveDomain(ctx, d.registryService, domain, d.resolver)

	if err != nil {
		return nil, err
	}

	ctx = clienturlctx.WithClientURL(ctx, u)

	translateNSE(nse, interdomain.Target)

	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}
