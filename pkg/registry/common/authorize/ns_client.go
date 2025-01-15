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

// Package authorize provides authz checks for incoming or returning connections.
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

type authorizeNSClient struct {
	policies     policiesList
	nsPathIDsMap *genericsync.Map[string, []string]
}

// NewNetworkServiceRegistryClient - returns a new authorization registry.NetworkServiceRegistryClient
// Authorize registry client checks spiffeID of NS.
func NewNetworkServiceRegistryClient(opts ...Option) registry.NetworkServiceRegistryClient {
	o := &options{
		resourcePathIDsMap: new(genericsync.Map[string, []string]),
	}

	for _, opt := range opts {
		opt(o)
	}

	return &authorizeNSClient{
		policies:     o.policies,
		nsPathIDsMap: o.resourcePathIDsMap,
	}
}

func (c *authorizeNSClient) Register(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	if len(c.policies) == 0 {
		return next.NetworkServiceRegistryClient(ctx).Register(ctx, ns, opts...)
	}

	path := grpcmetadata.PathFromContext(ctx)
	ctx = grpcmetadata.PathWithContext(ctx, path)

	var p peer.Peer
	opts = append(opts, grpc.Peer(&p))

	postponeCtxFunc := postpone.ContextWithValues(ctx)

	resp, err := next.NetworkServiceRegistryClient(ctx).Register(ctx, ns, opts...)
	if err != nil {
		return nil, err
	}

	if p != (peer.Peer{}) {
		ctx = peer.NewContext(ctx, &p)
	}

	path = grpcmetadata.PathFromContext(ctx)
	spiffeID := getSpiffeIDFromPath(ctx, path)
	rawMap := getRawMap(c.nsPathIDsMap)

	input := RegistryOpaInput{
		ResourceID:         spiffeID.String(),
		ResourceName:       resp.Name,
		ResourcePathIDsMap: rawMap,
		PathSegments:       path.PathSegments,
		Index:              path.Index,
	}
	if err := c.policies.check(ctx, input); err != nil {
		if _, load := c.nsPathIDsMap.Load(resp.Name); !load {
			unregisterCtx, cancelUnregister := postponeCtxFunc()
			defer cancelUnregister()

			if _, unregisterErr := next.NetworkServiceRegistryClient(ctx).Unregister(unregisterCtx, resp, opts...); unregisterErr != nil {
				err = errors.Wrapf(err, "nse unregistered with error: %s", unregisterErr.Error())
			}
		}

		return nil, err
	}

	c.nsPathIDsMap.Store(resp.Name, resp.PathIds)
	return resp, nil
}

func (c *authorizeNSClient) Find(ctx context.Context, query *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	return next.NetworkServiceRegistryClient(ctx).Find(ctx, query, opts...)
}

func (c *authorizeNSClient) Unregister(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	if len(c.policies) == 0 {
		return next.NetworkServiceRegistryClient(ctx).Unregister(ctx, ns, opts...)
	}

	path := grpcmetadata.PathFromContext(ctx)
	ctx = grpcmetadata.PathWithContext(ctx, path)

	var p peer.Peer
	opts = append(opts, grpc.Peer(&p))

	resp, err := next.NetworkServiceRegistryClient(ctx).Unregister(ctx, ns, opts...)
	if err != nil {
		return nil, err
	}

	if p != (peer.Peer{}) {
		ctx = peer.NewContext(ctx, &p)
	}

	spiffeID := getSpiffeIDFromPath(ctx, path)
	rawMap := getRawMap(c.nsPathIDsMap)

	input := RegistryOpaInput{
		ResourceID:         spiffeID.String(),
		ResourceName:       ns.Name,
		ResourcePathIDsMap: rawMap,
		PathSegments:       path.PathSegments,
		Index:              path.Index,
	}
	if err := c.policies.check(ctx, input); err != nil {
		return nil, err
	}

	c.nsPathIDsMap.Delete(ns.Name)
	return resp, nil
}
