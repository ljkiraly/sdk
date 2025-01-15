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

// Package authorize provides authorization checks for incoming or returning requests.
package authorize

import (
	"context"

	"github.com/edwarnicke/genericsync"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/ljkiraly/sdk/pkg/registry/common/grpcmetadata"
	"github.com/ljkiraly/sdk/pkg/registry/core/next"
	"github.com/ljkiraly/sdk/pkg/tools/postpone"
)

type authorizeNSEClient struct {
	policies      policiesList
	nsePathIDsMap *genericsync.Map[string, []string]
}

// NewNetworkServiceEndpointRegistryClient - returns a new authorization registry.NetworkServiceEndpointRegistryClient
// Authorize registry client checks path of NSE.
func NewNetworkServiceEndpointRegistryClient(opts ...Option) registry.NetworkServiceEndpointRegistryClient {
	o := &options{
		resourcePathIDsMap: new(genericsync.Map[string, []string]),
	}

	for _, opt := range opts {
		opt(o)
	}

	return &authorizeNSEClient{
		policies:      o.policies,
		nsePathIDsMap: o.resourcePathIDsMap,
	}
}

func (c *authorizeNSEClient) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	if len(c.policies) == 0 {
		return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, nse, opts...)
	}

	path := grpcmetadata.PathFromContext(ctx)

	ctx = grpcmetadata.PathWithContext(ctx, path)

	var p peer.Peer
	opts = append(opts, grpc.Peer(&p))

	postponeCtxFunc := postpone.ContextWithValues(ctx)

	resp, err := next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, nse, opts...)
	if err != nil {
		return nil, err
	}

	if p != (peer.Peer{}) {
		ctx = peer.NewContext(ctx, &p)
	}

	spiffeID := getSpiffeIDFromPath(ctx, path)
	rawMap := getRawMap(c.nsePathIDsMap)
	input := RegistryOpaInput{
		ResourceID:         spiffeID.String(),
		ResourceName:       resp.Name,
		ResourcePathIDsMap: rawMap,
		PathSegments:       path.PathSegments,
		Index:              path.Index,
	}
	if err := c.policies.check(ctx, input); err != nil {
		if _, load := c.nsePathIDsMap.Load(resp.Name); !load {
			unregisterCtx, cancelUnregister := postponeCtxFunc()
			defer cancelUnregister()

			if _, unregisterErr := next.NetworkServiceEndpointRegistryClient(ctx).Unregister(unregisterCtx, resp, opts...); unregisterErr != nil {
				err = errors.Wrapf(err, "nse unregistered with error: %s", unregisterErr.Error())
			}
		}

		return nil, err
	}

	c.nsePathIDsMap.Store(resp.Name, resp.PathIds)
	return resp, nil
}

func (c *authorizeNSEClient) Find(ctx context.Context, query *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, query, opts...)
}

func (c *authorizeNSEClient) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	if len(c.policies) == 0 {
		return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, nse, opts...)
	}

	path := grpcmetadata.PathFromContext(ctx)
	ctx = grpcmetadata.PathWithContext(ctx, path)

	var p peer.Peer
	opts = append(opts, grpc.Peer(&p))

	resp, err := next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, nse, opts...)
	if err != nil {
		return nil, err
	}

	if p != (peer.Peer{}) {
		ctx = peer.NewContext(ctx, &p)
	}

	spiffeID := getSpiffeIDFromPath(ctx, path)
	rawMap := getRawMap(c.nsePathIDsMap)
	input := RegistryOpaInput{
		ResourceID:         spiffeID.String(),
		ResourceName:       nse.Name,
		ResourcePathIDsMap: rawMap,
		PathSegments:       path.PathSegments,
		Index:              path.Index,
	}

	if err := c.policies.check(ctx, input); err != nil {
		return nil, err
	}

	c.nsePathIDsMap.Delete(nse.Name)
	return resp, nil
}
