HTTPSink
======================

Capture all requests that come into the server and be able to get those requests later for examination. Useful for faking API endpoints.

Using configuration or environment variables, you can set the foreign API endpoint in your code to use "localhost:8888" (or any port). Now, in tests, you can use httpsink to stand up an HTTP sink that will either just swallow up your requests or you can choose for it to yield a desired response.

## Features

- Set port or just use a random available port.
- Inspect the requests that come into the sink via the sink `/get?request_number=:request_number` endpoint where `request_number` is the zero-indexed request to come into the sink.
- Set the desired response from the sink.
- Set a capacity for the total number of requests that the sink will allow before it rejects them.
- Run multiple sinks at the same time.

Example:

```go
func TestSomeCode(t *testing.T) {
	// set up the sink (optionally set the port with the NewHTTPSinkOnAdder("localhost:8888"))
	hSync, _ := NewHTTPSink()
	defer hSync.Close()

	// set the behavior of the foreign API endpoint
	expectedBody := []byte(`{"key":"value"}`)
	hSync.SetResponse(&SimpleResponseWriter{StatusCode: http.StatusTeapot, Body: expectedBody})

	// make call to your code that in-turn makes a call to the foreign API endpoint
	err := sendAPIRequest()
	if err != nil{
	  t.Errorf("error sending api request - %s", err)
	}

	// optionaly verify that the foreign API got the right request if you like
	getURL := fmt.Sprintf("http://%s/get?request_number=0", hSync.Addr)
	getResp, _ := http.Get(getURL)
	defer getResp.Body.Close()

	capturedRequest := http.Request{}

	// ignoring the error because http.Request.Body does not play well with json.Decode
	// it works for this example, but you could also make a custom struct or use simplejson
	// if using a custom struct, just embedd *http.Request and mask with a Body interface{} property
	_ = json.NewDecoder(getResp.Body).Decode(&capturedRequest)

	if !strings.Contains(capturedRequest.URL.RawQuery, "some_key=some_value") {
		t.Errorf("captured request in sync did not capture proper query - %s", capturedRequest.URL.RawQuery)
	}
```
