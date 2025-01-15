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

package trace

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/ljkiraly/sdk/pkg/networkservice/core/trace/traceconcise"
	"github.com/ljkiraly/sdk/pkg/networkservice/core/trace/traceverbose"
	"github.com/ljkiraly/sdk/pkg/tools/log"
	"github.com/ljkiraly/sdk/pkg/tools/log/logruslogger"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type traceServer struct {
	verbose, concise, original networkservice.NetworkServiceServer
}

// NewNetworkServiceServer - wraps tracing around the supplied traced
func NewNetworkServiceServer(traced networkservice.NetworkServiceServer) networkservice.NetworkServiceServer {
	return &traceServer{
		verbose:  traceverbose.NewNetworkServiceServer(traced),
		concise:  traceconcise.NewNetworkServiceServer(traced),
		original: traced,
	}
}

func (t *traceServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if logrus.GetLevel() <= logrus.WarnLevel {
		if log.FromContext(ctx) == log.L() {
			ctx = log.WithLog(ctx, logruslogger.New(ctx))
		}
		return t.original.Request(ctx, request)
	}
	if logrus.GetLevel() >= logrus.DebugLevel {
		return t.verbose.Request(ctx, request)
	}

	return t.concise.Request(ctx, request)
}

func (t *traceServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	if logrus.GetLevel() <= logrus.WarnLevel {
		if log.FromContext(ctx) == log.L() {
			ctx = log.WithLog(ctx, logruslogger.New(ctx))
		}
		return t.original.Close(ctx, conn)
	}
	if logrus.GetLevel() >= logrus.DebugLevel {
		return t.verbose.Close(ctx, conn)
	}
	return t.concise.Close(ctx, conn)
}
