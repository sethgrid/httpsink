package httpsink_test

import (
	"encoding/json"
	"fmt"
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
	sink.Server.Addr = addr

	go func() {
		err = sink.ListenAndServe()
		require.NoError(t, err)
	}()

	<-time.After(time.Second)

	// insert 10 items
	for i := 1; i <= 10; i++ {
		r, err := http.NewRequest("POST", fmt.Sprintf("http://%s/", addr), strings.NewReader("hello!"))
		require.NoError(t, err)
		r.Header.Add("X-Test", fmt.Sprintf("index %d", i))

		res, err := http.DefaultClient.Do(r)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, res.StatusCode)
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
	assert.Equal(t, "/", request.URL)
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
	assert.Equal(t, "/", request.URL)
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

func randomPort() (int, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}

	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}
