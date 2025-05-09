// Copyright (c) 2021 Cisco and/or its affiliates.
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

//go:build !linux
// +build !linux

package recvfd

import (
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/ljkiraly/sdk/pkg/networkservice/common/null"
)

// NewServer - returns server chain element to recv FDs over the connection (if possible) for any Mechanism.Parameters[common.InodeURL]
// url of scheme 'inode'.
func NewServer() networkservice.NetworkServiceServer {
	return null.NewServer()
}
