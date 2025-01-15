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

//go:build linux
// +build linux

package recvfd_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/edwarnicke/grpcfd"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/ljkiraly/sdk/pkg/networkservice/chains/client"
	"github.com/ljkiraly/sdk/pkg/networkservice/common/begin"
	"github.com/ljkiraly/sdk/pkg/networkservice/common/mechanisms/recvfd"
	"github.com/ljkiraly/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/ljkiraly/sdk/pkg/networkservice/core/chain"
	"github.com/ljkiraly/sdk/pkg/networkservice/utils/checks/checkcontext"
	"github.com/ljkiraly/sdk/pkg/networkservice/utils/checks/checkcontextonreturn"
	"github.com/ljkiraly/sdk/pkg/networkservice/utils/inject/injecterror"
	"github.com/ljkiraly/sdk/pkg/tools/grpcfdutils"
	"github.com/ljkiraly/sdk/pkg/tools/grpcutils"
	"github.com/ljkiraly/sdk/pkg/tools/sandbox"
)

type checkRecvfdTestSuite struct {
	suite.Suite

	tempDir               string
	onFileClosedContexts  []context.Context
	onFileClosedCallbacks map[string]func()

	testClient networkservice.NetworkServiceClient
}

func (s *checkRecvfdTestSuite) SetupTest() {
	t := s.T()

	ctx, cancel := context.WithCancel(context.Background())

	t.Cleanup(func() {
		cancel()
		goleak.VerifyNone(t)
	})

	s.tempDir = t.TempDir()

	sock, err := os.Create(path.Join(s.tempDir, "test.sock"))
	s.Require().NoError(err)

	serveURL := &url.URL{Scheme: "unix", Path: sock.Name()}

	testChain := chain.NewNetworkServiceServer(
		begin.NewServer(),
		checkcontext.NewServer(t, func(t *testing.T, c context.Context) {
			injectErr := grpcfdutils.InjectOnFileReceivedCallback(c, func(fileName string, file *os.File) {
				runtime.SetFinalizer(file, func(file *os.File) {
					onFileClosedCallback, ok := s.onFileClosedCallbacks[fileName]
					if ok {
						onFileClosedCallback()
					}
				})
			})

			s.Require().NoError(injectErr)
		}),
		recvfd.NewServer())

	startServer(ctx, s, &testChain, serveURL)
	s.testClient = createClient(ctx, serveURL)
}

func TestRecvfd(t *testing.T) {
	suite.Run(t, new(checkRecvfdTestSuite))
}

func startServer(ctx context.Context, s *checkRecvfdTestSuite, testServerChain *networkservice.NetworkServiceServer, serveURL *url.URL) {
	grpcServer := grpc.NewServer(grpc.Creds(grpcfd.TransportCredentials(insecure.NewCredentials())))
	networkservice.RegisterNetworkServiceServer(grpcServer, *testServerChain)

	errCh := grpcutils.ListenAndServe(ctx, serveURL, grpcServer)

	s.Require().Len(errCh, 0)
}

func createClient(ctx context.Context, u *url.URL) networkservice.NetworkServiceClient {
	return client.NewClient(
		ctx,
		client.WithClientURL(sandbox.CloneURL(u)),
		client.WithDialOptions(grpc.WithTransportCredentials(
			grpcfd.TransportCredentials(insecure.NewCredentials())),
		),
		client.WithDialTimeout(time.Second),
		client.WithoutRefresh(),
		client.WithAdditionalFunctionality(sendfd.NewClient()))
}

func createFile(s *checkRecvfdTestSuite, fileName string) (inodeURLStr string, fileClosedContext context.Context, cancelFunc func()) {
	// #nosec
	f, err := os.Create(fileName)
	s.Require().NoErrorf(err, "Failed to create and open a file: %v", err)

	err = f.Close()
	s.Require().NoErrorf(err, "Failed to close file: %v", err)

	fileClosedContext, cancelFunc = context.WithCancel(context.Background())

	inodeURL, err := grpcfd.FilenameToURL(fileName)
	s.Require().NoError(err)

	return inodeURL.String(), fileClosedContext, cancelFunc
}

func (s *checkRecvfdTestSuite) TestRecvfdClosesSingleFile() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	testFileName := path.Join(s.tempDir, "TestRecvfdClosesSingleFile.test")

	inodeURLStr, fileClosedContext, cancelFunc := createFile(s, testFileName)

	s.onFileClosedCallbacks = map[string]func(){
		inodeURLStr: cancelFunc,
	}

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{
				Cls:  cls.LOCAL,
				Type: kernel.MECHANISM,
				Parameters: map[string]string{
					common.InodeURL: "file:" + testFileName,
				},
			},
		},
	}

	conn, err := s.testClient.Request(ctx, request)
	s.Require().NoError(err)

	_, err = s.testClient.Close(ctx, conn)
	s.Require().NoError(err)

	s.Require().Eventually(func() bool {
		runtime.GC()
		return fileClosedContext.Err() != nil
	}, time.Second, time.Millisecond*100)
}

func (s *checkRecvfdTestSuite) TestRecvfdClosesMultipleFiles() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	const numFiles = 3
	s.onFileClosedContexts = make([]context.Context, numFiles)
	s.onFileClosedCallbacks = make(map[string]func(), numFiles)

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: make([]*networkservice.Mechanism, numFiles),
	}

	var filePath string
	for i := 0; i < numFiles; i++ {
		filePath = path.Join(s.tempDir, fmt.Sprintf("TestRecvfdClosesMultipleFiles.test%d", i))

		inodeURLStr, fileClosedContext, cancelFunc := createFile(s, filePath)
		s.onFileClosedCallbacks[inodeURLStr] = cancelFunc
		s.onFileClosedContexts[i] = fileClosedContext

		request.MechanismPreferences = append(request.MechanismPreferences,
			&networkservice.Mechanism{
				Cls:  cls.LOCAL,
				Type: kernel.MECHANISM,
				Parameters: map[string]string{
					common.InodeURL: "file:" + filePath,
				},
			})
	}

	conn, err := s.testClient.Request(ctx, request)
	s.Require().NoError(err)

	_, err = s.testClient.Close(ctx, conn)
	s.Require().NoError(err)

	for i := range s.onFileClosedContexts {
		onClosedFileCtx := s.onFileClosedContexts[i]
		s.Require().Eventually(func() bool {
			runtime.GC()
			return onClosedFileCtx.Err() != nil
		}, time.Second, time.Millisecond*100)
	}
}

func TestRecvfdDoesntWaitForAnyFilesOnRequestsFromBegin(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)

	t.Cleanup(func() {
		cancel()
		goleak.VerifyNone(t)
	})

	eventFactoryCh := make(chan begin.EventFactory, 1)
	var once sync.Once
	// Create a server
	server := chain.NewNetworkServiceServer(
		begin.NewServer(),
		checkcontextonreturn.NewServer(t, func(t *testing.T, ctx context.Context) {
			once.Do(func() {
				eventFactoryCh <- begin.FromContext(ctx)
				close(eventFactoryCh)
			})
		}),
		recvfd.NewServer(),
		injecterror.NewServer(
			injecterror.WithError(errors.New("error")),
			injecterror.WithRequestErrorTimes(1),
			injecterror.WithCloseErrorTimes(1)),
	)

	tempDir := t.TempDir()
	sock, err := os.Create(filepath.Clean(path.Join(tempDir, "test.sock")))
	require.NoError(t, err)

	serveURL := &url.URL{Scheme: "unix", Path: sock.Name()}
	grpcServer := grpc.NewServer(grpc.Creds(grpcfd.TransportCredentials(insecure.NewCredentials())))
	networkservice.RegisterNetworkServiceServer(grpcServer, server)
	errCh := grpcutils.ListenAndServe(ctx, serveURL, grpcServer)
	require.Len(t, errCh, 0)

	// Create a client
	c := createClient(ctx, serveURL)

	// Create a file to send
	testFileName := filepath.Clean(path.Join(tempDir, "TestRecvfdDoesntWaitForAnyFilesOnRequestsFromBegin.test"))
	f, err := os.Create(testFileName)
	require.NoErrorf(t, err, "Failed to create and open a file: %v", err)
	err = f.Close()
	require.NoErrorf(t, err, "Failed to close file: %v", err)

	// Create a request
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "id",
			Mechanism: &networkservice.Mechanism{
				Cls:  cls.LOCAL,
				Type: kernel.MECHANISM,
				Parameters: map[string]string{
					common.InodeURL: "file:" + testFileName,
				},
			},
		},
	}

	// Make the first request from the client to send files
	conn, err := c.Request(ctx, request)
	require.NoError(t, err)
	request.Connection = conn.Clone()

	// Make the second request that return an error.
	// It should make recvfd close all the files.
	_, err = c.Request(ctx, request)
	require.Error(t, err)

	// Send Close. Recvfd shouldn't freeze trying to read files
	// from the client because we send Close from begin.
	eventFactory := <-eventFactoryCh
	ch := eventFactory.Close()
	err = <-ch
	require.NoError(t, err)
}
