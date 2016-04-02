package httpsink

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func TestSyncRetrieval(t *testing.T) {
	hSync, _ := NewHTTPSink()
	hSync.Capacity = 10
	defer hSync.Close()
	go hSync.StartHTTP()
	setURL := fmt.Sprintf("http://%s/some/url?some_key=some_value&some_other_key=some_other_value", hSync.Addr)
	// setResp, err := http.Get(setURL)
	setResp, err := http.PostForm(setURL, url.Values{"key": {"Value"}, "id": {"123"}})

	if err != nil {
		t.Errorf("unable to GET on set sync, %v", err)
		return
	}

	if got, want := setResp.StatusCode, http.StatusCreated; got != want {
		t.Errorf("incorrect status code. got %d, want %d", got, want)
	}

	getURL := fmt.Sprintf("http://%s/get?request_number=0", hSync.Addr)
	getResp, err := http.Get(getURL)

	if err != nil {
		t.Errorf("unable to GET on get sync, %v", err)
		return
	}

	if got, want := getResp.StatusCode, http.StatusOK; got != want {
		t.Errorf("incorrect status code. got %d, want %d", got, want)
	}

	capturedRequest := RequestMask{}

	err = json.NewDecoder(getResp.Body).Decode(&capturedRequest)
	if err != nil {
		t.Errorf("response body decode error - %v", err)
		t.Errorf("captured request - %+v", capturedRequest)
	}

	if capturedRequest.URL == nil {
		t.Errorf("captured request url property should not be nil")
		return
	}

	if !strings.Contains(capturedRequest.URL.RawQuery, "some_key=some_value") {
		t.Errorf("captured request in sync did not capture proper query - %s", capturedRequest.URL.RawQuery)
	}
}

func TestSyncRetrievalIndexError(t *testing.T) {
	hSync, _ := NewHTTPSink()
	defer hSync.Close()
	go hSync.StartHTTP()
	setURL := fmt.Sprintf("http://%s/some/url?some_key=some_value&some_other_key=some_other_value", hSync.Addr)
	setResp, err := http.Get(setURL)

	if err != nil {
		t.Errorf("unable to GET on set sync, %v", err)
		return
	}

	if got, want := setResp.StatusCode, http.StatusCreated; got != want {
		t.Errorf("incorrect status code. got %d, want %d", got, want)
	}

	// index of 1, but should be 0 indexed
	getURL := fmt.Sprintf("http://%s/get?request_number=1", hSync.Addr)
	getResp, err := http.Get(getURL)

	if err != nil {
		t.Errorf("unable to GET on get sync, %v", err)
		return
	}

	if got, want := getResp.StatusCode, http.StatusBadRequest; got != want {
		t.Errorf("incorrect status code. got %d, want %d", got, want)
	}

	resp, err := ioutil.ReadAll(getResp.Body)
	if err != nil {
		t.Error("response body read error", err)
	}
	if !strings.Contains(string(resp), "not valid") {
		t.Errorf("invalid response content. got '%v', want something with 'not valid'", string(resp))
	}
}

func TestSyncRetrievalCapacity(t *testing.T) {
	hSync, _ := NewHTTPSink()
	defer hSync.Close()
	hSync.Capacity = 1
	go hSync.StartHTTP()
	setURL := fmt.Sprintf("http://%s/some/url?some_key=some_value&some_other_key=some_other_value", hSync.Addr)
	setResp, err := http.Get(setURL)

	if err != nil {
		t.Errorf("unable to GET on set sync, %v", err)
		return
	}

	if got, want := setResp.StatusCode, http.StatusCreated; got != want {
		t.Errorf("incorrect status code. got %d, want %d", got, want)
	}

	// another call setting data
	setURL = fmt.Sprintf("http://%s/some/url?some_key=some_value&some_other_key=some_other_value", hSync.Addr)
	setResp, err = http.Get(setURL)

	if err != nil {
		t.Errorf("unable to GET on set sync, %v", err)
		return
	}

	if got, want := setResp.StatusCode, http.StatusGone; got != want {
		t.Errorf("incorrect status code. got %d, want %d", got, want)
	}
}

func TestSetResponse(t *testing.T) {
	hSync, _ := NewHTTPSink()
	defer hSync.Close()

	expectedBody := []byte(`{"key":"value"}`)
	hSync.SetResponse(&SimpleResponseWriter{StatusCode: http.StatusTeapot, Body: expectedBody})

	go hSync.StartHTTP()
	setURL := fmt.Sprintf("http://%s/some/url", hSync.Addr)
	resp, err := http.Get(setURL)

	if err != nil {
		t.Errorf("unable to GET on set sync, %v", err)
		return
	}

	if got, want := resp.StatusCode, http.StatusTeapot; got != want {
		t.Errorf("incorrect status code. got %d, want %d", got, want)
	}

	body, _ := ioutil.ReadAll(resp.Body)
	if got, want := string(body), string(expectedBody); got != want {
		t.Errorf("incorrect response body. got %s, want %s", got, want)
	}
}

func TestClearRequests(t *testing.T) {
	hSync, _ := NewHTTPSink()
	defer hSync.Close()
	hSync.Capacity = 1
	go hSync.StartHTTP()

	// clear requests
	clearURL := fmt.Sprintf("http://%s/clear", hSync.Addr)
	clearResp, err := http.Get(clearURL)

	if err != nil {
		t.Errorf("unable to clear requests, %v", err)
		return
	}

	if got, want := clearResp.StatusCode, http.StatusNoContent; got != want {
		t.Errorf("incorrect status code. got %d, want %d", got, want)
	}
}

func TestReturnBodyOnly(t *testing.T) {
	hSync, _ := NewHTTPSink()
	defer hSync.Close()
	hSync.Capacity = 1
	hSync.BodyOnly = true
	go hSync.StartHTTP()

	data := url.Values{"test": {"body only"}}
	setStr := fmt.Sprintf("http://%s/something", hSync.Addr)

	client := &http.Client{}
	r, _ := http.NewRequest("GET", setStr, bytes.NewBufferString(data.Encode()))
	_, err := client.Do(r)
	if err != nil {
		t.Errorf("fail to call set %v", err)
		return
	}

	getURL := fmt.Sprintf("http://%s/get?request_number=0", hSync.Addr)
	getResp, err := http.Get(getURL)
	if err != nil {
		t.Errorf("fail to call get %v", err)
		return
	}
	body, _ := ioutil.ReadAll(getResp.Body)

	regex, _ := regexp.Compile("body")
	if !regex.MatchString(string(body)) {
		t.Errorf("Expected body to contain 'body' but got %s", string(body))
	}
}
