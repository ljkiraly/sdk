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

package checkmsg_test

import (
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"

	"github.com/ljkiraly/sdk/pkg/tools/dnsutils/checkmsg"
)

type responseWriter struct {
	dns.ResponseWriter
	Response *dns.Msg
}

func (r *responseWriter) WriteMsg(m *dns.Msg) error {
	r.Response = m
	return nil
}

func TestCheckMsgHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	handler := checkmsg.NewDNSHandler()
	rw := &responseWriter{}
	handler.ServeDNS(ctx, rw, nil)
	require.NotEqual(t, rw.Response.Rcode, dns.RcodeSuccess)
}
