package httpsink_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sendgrid/httpsink"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSink(t *testing.T) {
	sink := httpsink.New()

	port, err := randomPort()
	require.NoError(t, err)

	t.Logf("using port %d", port)
	addr := fmt.Sprintf("localhost:%d", port)
	sink.Addr = addr

	go func() {
		err = sink.ListenAndServe()
		require.NoError(t, err)
	}()

	<-time.After(time.Second)

	// insert 10 items
	for i := 1; i <= 10; i++ {
		r, err := http.NewRequest("POST", fmt.Sprintf("http://%s/blah/something", addr), strings.NewReader("hello!"))
		require.NoError(t, err)
		r.Header.Add("X-Test", fmt.Sprintf("index %d", i))

		res, err := http.DefaultClient.Do(r)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	}

	// get first one
	r, err := http.NewRequest("GET", fmt.Sprintf("http://%s/request/1", addr), nil)
	require.NoError(t, err)

	res, err := http.DefaultClient.Do(r)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	request := httpsink.Request{}
	err = json.NewDecoder(res.Body).Decode(&request)
	require.NoError(t, err)

	assert.Equal(t, "POST", request.Method)
	assert.Equal(t, "/blah/something", request.URL)
	assert.Equal(t, "hello!", string(request.Body))
	assert.Equal(t, []string{"index 1"}, request.Headers["X-Test"])

	// get last one
	r, err = http.NewRequest("GET", fmt.Sprintf("http://%s/requests/last", addr), nil)
	require.NoError(t, err)

	res, err = http.DefaultClient.Do(r)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	request = httpsink.Request{}
	err = json.NewDecoder(res.Body).Decode(&request)
	require.NoError(t, err)

	assert.Equal(t, "POST", request.Method)
	assert.Equal(t, "/blah/something", request.URL)
	assert.Equal(t, "hello!", string(request.Body))
	assert.Equal(t, []string{"index 10"}, request.Headers["X-Test"])

	// clear requests
	r, err = http.NewRequest("DELETE", fmt.Sprintf("http://%s/requests", addr), nil)
	require.NoError(t, err)
	res, err = http.DefaultClient.Do(r)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, res.StatusCode)

	// make sure it's cleared
	r, err = http.NewRequest("GET", fmt.Sprintf("http://%s/request/1", addr), nil)
	require.NoError(t, err)
	res, err = http.DefaultClient.Do(r)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)
}

func TestMail(t *testing.T) {
	sink := httpsink.New()

	port, err := randomPort()
	require.NoError(t, err)

	t.Logf("using port %d", port)
	addr := fmt.Sprintf("localhost:%d", port)
	sink.Addr = addr

	go func() {
		err = sink.ListenAndServe()
		require.NoError(t, err)
	}()

	<-time.After(time.Second)

	// v2 in x-smtpapi header
	r, err := http.NewRequest("GET", fmt.Sprintf("http://%s/mail.send.json", addr), nil)
	require.NoError(t, err)
	r.Header.Add("X-SMTPAPI", `{"to":["test1@test.com","all@test.com"]}`)

	res, err := http.DefaultClient.Do(r)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	// v2 in URL
	r, err = http.NewRequest("GET", fmt.Sprintf("http://%s/mail.send.json?to[]=test2@test.com&to[]=all@test.com", addr), nil)
	require.NoError(t, err)

	res, err = http.DefaultClient.Do(r)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	// v2 in URL (single)
	r, err = http.NewRequest("GET", fmt.Sprintf("http://%s/mail.send.json?to=test3@test.com", addr), nil)
	require.NoError(t, err)

	res, err = http.DefaultClient.Do(r)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	// v3
	r, err = http.NewRequest("GET", fmt.Sprintf("http://%s/mail.send.json", addr), nil)
	r.Header.Add("X-SMTPAPI", `{"personalizations":[{"to":[{"email":"test4@test.com"},{"email":"all@test.com"}]}]}`)
	require.NoError(t, err)

	res, err = http.DefaultClient.Do(r)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	data := make(map[string]interface{})
	r, err = http.NewRequest("GET", fmt.Sprintf("http://%s/requests/recipient/test1@test.com", addr), nil)
	assert.NoError(t, err)

	res, err = http.DefaultClient.Do(r)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	err = json.NewDecoder(res.Body).Decode(&data)
	assert.NoError(t, err)
	assert.Len(t, data["requests"], 1)

	data = make(map[string]interface{})
	r, err = http.NewRequest("GET", fmt.Sprintf("http://%s/requests/recipient/test2@test.com", addr), nil)
	assert.NoError(t, err)

	res, err = http.DefaultClient.Do(r)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	err = json.NewDecoder(res.Body).Decode(&data)
	assert.NoError(t, err)
	assert.Len(t, data["requests"], 1)

	data = make(map[string]interface{})
	r, err = http.NewRequest("GET", fmt.Sprintf("http://%s/requests/recipient/test3@test.com", addr), nil)
	assert.NoError(t, err)

	res, err = http.DefaultClient.Do(r)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	err = json.NewDecoder(res.Body).Decode(&data)
	assert.NoError(t, err)
	assert.Len(t, data["requests"], 1)

	data = make(map[string]interface{})
	r, err = http.NewRequest("GET", fmt.Sprintf("http://%s/requests/recipient/test4@test.com", addr), nil)
	assert.NoError(t, err)

	res, err = http.DefaultClient.Do(r)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	err = json.NewDecoder(res.Body).Decode(&data)
	assert.NoError(t, err)
	assert.Len(t, data["requests"], 1)

	data = make(map[string]interface{})
	r, err = http.NewRequest("GET", fmt.Sprintf("http://%s/requests/recipient/all@test.com", addr), nil)
	assert.NoError(t, err)

	res, err = http.DefaultClient.Do(r)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	err = json.NewDecoder(res.Body).Decode(&data)
	assert.NoError(t, err)
	assert.Len(t, data["requests"], 3)

	data = make(map[string]interface{})
	r, err = http.NewRequest("GET", fmt.Sprintf("http://%s/requests/recipient/unknown@test.com", addr), nil)
	assert.NoError(t, err)

	res, err = http.DefaultClient.Do(r)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	err = json.NewDecoder(res.Body).Decode(&data)
	assert.NoError(t, err)
	assert.Len(t, data["requests"], 0)
}

func TestProxy(t *testing.T) {
	sink := httpsink.New()

	reqCh := make(chan *http.Request)

	port, err := randomPort()
	require.NoError(t, err)
	sinkAddr := fmt.Sprintf("localhost:%d", port)
	sink.Addr = sinkAddr

	go func() {
		err = sink.ListenAndServe()
		require.NoError(t, err)
	}()

	<-time.After(time.Second)

	go func() {
		port, err := randomPort()
		require.NoError(t, err)

		addr := fmt.Sprintf("localhost:%d", port)
		sink.Proxy = fmt.Sprintf("http://%s", addr)

		http.HandleFunc("/mypath", func(rw http.ResponseWriter, r *http.Request) {
			reqCh <- r
		})

		err = http.ListenAndServe(addr, nil)
		require.NoError(t, err)
	}()

	<-time.After(time.Second)

	r, err := http.NewRequest("POST", fmt.Sprintf("http://%s/mypath", sinkAddr), strings.NewReader("wubba lubba dub dub"))
	require.NoError(t, err)
	r.Header.Add("X-Test", "hello")

	res, err := http.DefaultClient.Do(r)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	t.Log("waiting for request...")
	request := <-reqCh

	assert.Equal(t, "/mypath", request.URL.Path)
	assert.Equal(t, "POST", request.Method, "POST")
	assert.Equal(t, "hello", request.Header.Get("X-Test"))

	body := bytes.NewBuffer(nil)
	io.Copy(body, request.Body)
	defer request.Body.Close()

	assert.Equal(t, "wubba lubba dub dub", body.String())
}

func randomPort() (int, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}

	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}
