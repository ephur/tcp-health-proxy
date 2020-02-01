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
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckHealth(t *testing.T) {
	logSetup("panic")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "ok")
	}))
	c := make(chan bool, 1)

	tsBad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))

	defer ts.Close()
	defer tsBad.Close()
	defer close(c)

	var result bool

	checkHealth(c, ts.URL, regexp.MustCompile("ok"))
	result = <-c
	assert.True(t, result, "with provided body, result should be true")

	checkHealth(c, ts.URL, regexp.MustCompile("notok"))
	result = <-c
	assert.False(t, result, "with provided body, result should be false")

	checkHealth(c, tsBad.URL, regexp.MustCompile("ok"))
	result = <-c
	assert.False(t, result, "http error code sent, result should be false")

}
