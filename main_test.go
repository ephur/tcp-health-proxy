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
