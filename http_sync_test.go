package httpsync

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

func TestSyncRetrieval(t *testing.T) {
	hSync, _ := NewHttpSync()
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

	getURL := fmt.Sprintf("http://%s/get?idx=0", hSync.Addr)
	getResp, err := http.Get(getURL)

	if err != nil {
		t.Errorf("unable to GET on get sync, %v", err)
		return
	}

	if got, want := getResp.StatusCode, http.StatusOK; got != want {
		t.Errorf("incorrect status code. got %d, want %d", got, want)
	}

	capturedRequest := http.Request{}

	err = json.NewDecoder(getResp.Body).Decode(&capturedRequest)
	if err != nil {
		// not sure what is going on here. It is
		// populating the struct, but still complains
		// about using a ReadCloser().
		t.Log("response body decode error - %v", err, capturedRequest)
	}

	if !strings.Contains(capturedRequest.URL.RawQuery, "some_key=some_value") {
		t.Errorf("captured request in sync did not capture proper query - %s", capturedRequest.URL.RawQuery)
	}

}

func TestSyncRetrievalIndexError(t *testing.T) {
	hSync, _ := NewHttpSync()
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
	getURL := fmt.Sprintf("http://%s/get?idx=1", hSync.Addr)
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
		t.Errorf("response body read error", err)
	}
	if !strings.Contains(string(resp), "not valid") {
		t.Errorf("invalid response content. got '%v', want something with 'not valid'", string(resp))
	}
}

func TestSyncRetrievalCapacity(t *testing.T) {
	hSync, _ := NewHttpSync()
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
