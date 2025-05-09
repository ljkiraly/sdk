// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023-2024 Cisco and/or its affiliates.
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

// Package discoverforwarder discovers forwarder from the registry.
package discoverforwarder

import (
	"context"
	"net/url"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"

	"github.com/ljkiraly/sdk/pkg/networkservice/core/next"
	"github.com/ljkiraly/sdk/pkg/tools/clienturlctx"
	"github.com/ljkiraly/sdk/pkg/tools/log"
	"github.com/ljkiraly/sdk/pkg/tools/matchutils"
)

type discoverForwarderServer struct {
	nseClient            registry.NetworkServiceEndpointRegistryClient
	nsClient             registry.NetworkServiceRegistryClient
	forwarderServiceName string
	nsmgrURL             string
}

// NewServer creates new instance of discoverforwarder networkservice.NetworkServiceServer.
// Requires not nil nseClient.
// Requires not nil nsClient.
func NewServer(nsClient registry.NetworkServiceRegistryClient, nseClient registry.NetworkServiceEndpointRegistryClient, opts ...Option) networkservice.NetworkServiceServer {
	if nseClient == nil {
		panic("nseClient can not be nil")
	}
	if nsClient == nil {
		panic("nsClient can not be nil")
	}

	var result = &discoverForwarderServer{
		nseClient:            nseClient,
		nsClient:             nsClient,
		forwarderServiceName: "forwarder",
	}

	for _, opt := range opts {
		opt(result)
	}

	return result
}

func (d *discoverForwarderServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	var forwarderName = loadForwarderName(ctx)
	var logger = log.FromContext(ctx).WithField("discoverForwarderServer", "request")

	if request.GetConnection().State == networkservice.State_RESELECT_REQUESTED {
		forwarderName = ""
	}

	ns, err := d.discoverNetworkService(ctx, request.GetConnection().GetNetworkService(), request.GetConnection().GetPayload())
	if err != nil {
		return nil, err
	}

	stream, err := d.nseClient.Find(ctx, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			NetworkServiceNames: []string{
				d.forwarderServiceName,
			},
			Name: forwarderName,
			Url:  d.nsmgrURL,
		},
	})
	if err != nil {
		logger.Errorf("can not open registry nse stream by networkservice. Error: %v", err.Error())
		return nil, errors.Wrapf(err, "failed to find %s on %s", d.forwarderServiceName, d.nsmgrURL)
	}

	nses := d.matchForwarders(request.Connection.GetLabels(), ns, registry.ReadNetworkServiceEndpointList(stream))
	if len(nses) == 0 {
		if forwarderName != "" {
			return nil, errors.Errorf("forwarder %v is not available", forwarderName)
		}
		return nil, errors.New("no candidates found")
	}

	if forwarderName == "" && request.GetConnection().GetState() != networkservice.State_RESELECT_REQUESTED {
		segments := request.Connection.GetPath().GetPathSegments()
		if pathIndex := int(request.Connection.GetPath().Index); len(segments) > pathIndex+1 {
			for i, candidate := range nses {
				if candidate.Name == segments[pathIndex+1].GetName() {
					nses[0], nses[i] = nses[i], nses[0]
					break
				}
			}
		}
	}

	var candidatesErr = errors.New("all forwarders have failed")

	// TODO: Should we consider about load balancing?
	// https://github.com/ljkiraly/sdk/issues/790
	for i, candidate := range nses {
		u, err := url.Parse(candidate.Url)
		if err != nil {
			logger.Errorf("can not parse forwarder=%v url=%v error=%v", candidate.Name, candidate.Url, err.Error())
			continue
		}

		resp, err := next.Server(ctx).Request(clienturlctx.WithClientURL(ctx, u), request.Clone())
		if err == nil {
			if forwarderName == "" {
				storeForwarderName(ctx, candidate.GetName())
			}
			return resp, nil
		}
		logger.Errorf("forwarder=%v url=%v returned error=%v", candidate.Name, candidate.Url, err.Error())
		candidatesErr = errors.Wrapf(candidatesErr, "%v. An error during select forwarder %v --> %v", i, candidate.Name, err.Error())
	}

	return nil, candidatesErr
}

func (d *discoverForwarderServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	// Unlike Request, Close method should always call next element in chain
	// to make sure we clear resources in the current app.

	var forwarderName = loadForwarderName(ctx)

	if forwarderName == "" {
		segments := conn.GetPath().GetPathSegments()
		if pathIndex := int(conn.GetPath().Index); len(segments) > pathIndex+1 {
			forwarderName = segments[pathIndex+1].GetName()
		}
	}

	var logger = log.FromContext(ctx).WithField("discoverForwarderServer", "request")
	if forwarderName == "" {
		logger.Error("connection doesn't have forwarder")
		return next.Server(ctx).Close(ctx, conn)
	}

	stream, err := d.nseClient.Find(ctx, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name: forwarderName,
			Url:  d.nsmgrURL,
		},
	})
	if err != nil {
		logger.Errorf("can not open registry nse stream by forwarder name %v. Error: %v", forwarderName, err.Error())
		return next.Server(ctx).Close(ctx, conn)
	}

	nses := registry.ReadNetworkServiceEndpointList(stream)
	if len(nses) == 0 {
		logger.Errorf("forwarder is not found: %v", forwarderName)
		return next.Server(ctx).Close(ctx, conn)
	}

	u, err := url.Parse(nses[0].Url)
	if err != nil {
		logger.Errorf("can not parse forwarder url %v: %v", nses[0].Url, err.Error())
		return next.Server(ctx).Close(ctx, conn)
	}

	ctx = clienturlctx.WithClientURL(ctx, u)
	return next.Server(ctx).Close(ctx, conn)
}

func (d *discoverForwarderServer) matchForwarders(nsLabels map[string]string, ns *registry.NetworkService, nses []*registry.NetworkServiceEndpoint) []*registry.NetworkServiceEndpoint {
	var result []*registry.NetworkServiceEndpoint

	if len(ns.GetMatches()) == 0 {
		return nses
	}

	for _, match := range ns.GetMatches() {
		if !matchutils.IsSubset(nsLabels, match.GetSourceSelector(), nsLabels) {
			continue
		}

		var matchLabels = match.GetMetadata().GetLabels()
		if matchLabels == nil {
			matchLabels = map[string]string{
				"p2p": "true",
			}
		}
		for _, nse := range nses {
			var forwarderLabels = nse.GetNetworkServiceLabels()[d.forwarderServiceName]
			if forwarderLabels == nil {
				continue
			}
			if matchutils.IsSubset(forwarderLabels.Labels, matchLabels, nsLabels) {
				result = append(result, nse)
			}
		}

		if match.Fallthrough && len(result) == 0 {
			continue
		}

		break
	}

	return result
}

func (d *discoverForwarderServer) discoverNetworkService(ctx context.Context, name, payload string) (*registry.NetworkService, error) {
	query := &registry.NetworkServiceQuery{
		NetworkService: &registry.NetworkService{
			Name:    name,
			Payload: payload,
		},
	}

	nsRespStream, err := d.nsClient.Find(ctx, query)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find %s", query.String())
	}

	nsList := registry.ReadNetworkServiceList(nsRespStream)
	for _, ns := range nsList {
		if ns.Name == name {
			return ns, nil
		}
	}

	return nil, errors.Errorf("network service %v is not found", name)
}
