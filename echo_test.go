// Copyright 2020 Richard Maynard (richard.maynard@gmail.com)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bufio"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type EchoTestSuite struct {
	suite.Suite
	EchoServer *Echo
}

func (suite *EchoTestSuite) SetupSuite() {
	suite.EchoServer = NewEcho("127.0.0.1", "1580")
	logSetup("panic")
}

func (suite *EchoTestSuite) TearDownSuite() {
	suite.EchoServer.Down()
	for i := true; i != true; {
		i = suite.EchoServer.closing
	}
}

func (suite *EchoTestSuite) TestMultiEchoUp() {
	// multiple calls to server up should return the references to the same object
	p := suite.EchoServer.Up()
	assert.IsType(suite.T(), Echo{}, *p, "p should be an echo object")

	// wait for server startup to complete
	for i := true; i != false; {
		i = suite.EchoServer.starting
	}

	// Assign the echoserver to another var, and make sure it's the same
	e := suite.EchoServer.Up()
	assert.Equal(suite.T(), p, e, "multiple up calls should return the same echo server")
}

func (suite *EchoTestSuite) TestMultiEchoDown() {
	// multiple calls to server down should succeed, and return references to the same object
	// this also verifies the service properly/cleanly shuts down
	suite.EchoServer.Up()

	// wait for server to start
	for i := true; i != false; {
		i = suite.EchoServer.starting
	}

	e := suite.EchoServer.Down()
	// wait for server to stop
	for i := true; i != true; {
		i = suite.EchoServer.closing
	}
	e2 := suite.EchoServer.Down()

	assert.Equal(suite.T(), e, e2, "multiple up calls should return the same echo server")
}

func (suite *EchoTestSuite) TestEchoReply() {
	// tests to ensure the echo service responds
	suite.EchoServer.Up()
	// text must end with newline
	testText := "echo server test\n"

	// wait for server to start
	for i := true; i != false; {
		i = suite.EchoServer.starting
	}

	conn, err := net.Dial("tcp", "127.0.0.1:1580")
	assert.Nil(suite.T(), err)
	defer conn.Close()
	fmt.Fprintf(conn, testText+"\n")
	message, err := bufio.NewReader(conn).ReadString('\n')
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), testText, message)
}

func TestEchoSuite(t *testing.T) {
	suite.Run(t, new(EchoTestSuite))
}
