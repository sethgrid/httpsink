HTTPSink
======================

Capture all requests that come into the server and be able to get those requests later for examination. Useful for faking API endpoints.

Using configuration or environment variables, you can set the foreign API endpoint in your code to use "localhost:8888" (or any port). Now, in tests, you can use httpsink to stand up an HTTP sink that will either just swallow up your requests or you can choose for it to yield a desired response.

## Features

- Set port or just use a random available port.
- Inspect the requests that come into the sink via the sink `/get?request_number=:request_number` endpoint where `request_number` is the zero-indexed request to come into the sink.
- Set the desired response from the sink.
- Set a capacity for the total number of requests that the sink will allow before it rejects them.
- Clear stored requests
- Specify if you want to get the body of the request only
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

	// Set the number of requests that you want to capture.
	// If unset (ie, 0), the sync will never store results and always give back your default response.
	// If set, the server will only accept at max that many requests before returning errors.
	// Capacity must be set for httpsink to store the result for later retrieval.
	hSync.Capacity = 5

	// make call to your code that in-turn makes a call to the foreign API endpoint
	err := sendAPIRequest()
	if err != nil{
	  t.Errorf("error sending api request - %s", err)
	}

	// optionaly verify that the foreign API got the right request if you like
	// hSync.Capacity will have to have been set for this to work.
	getURL := fmt.Sprintf("http://%s/get?request_number=0", hSync.Addr)
	getResp, _ := http.Get(getURL)
	defer getResp.Body.Close()

    // cannot marshal into http.Request due to Body being a ReadCloser()
    // hSync.RequestMask allows us to get access to the Body
	capturedRequest := hSync.RequestMask{}

	// ignoring the error because http.Request.Body does not play well with json.Decode
	// it works for this example, but you could also make a custom struct or use simplejson
	// if using a custom struct, just embedd *http.Request and mask with a Body interface{} property
	_ = json.NewDecoder(getResp.Body).Decode(&capturedRequest)

	if !strings.Contains(capturedRequest.URL.RawQuery, "some_key=some_value") {
		t.Errorf("captured request in sync did not capture proper query - %s", capturedRequest.URL.RawQuery)
	}

	// if you need to clear the stored requests to make room for the next call/test,
	// use the following endpoint. It will return a 204
	clearURL := fmt.Sprintf("http://%s/clear", hSync.Addr)
	http.Get(clearURL)

	// to get the actual request body, set the following hSync config and call get normally
	hSync.BodyOnly = true
	bodyResp, _ := http.Get(getURL)
	defer bodyResp.Body.Close()
}
```

The response that come back from captured responses at `http://$hSync.Addr/get?request_number=x` will look like the following, and can be marshalled into `httpsink.RequestMask{}`:
```
{"Method":"POST","URL":{"Scheme":"","Opaque":"","User":null,"Host":"","Path":"/some/url","RawPath":"","RawQuery":"some_key=some_value\u0026some_other_key=some_other_value","Fragment":""},"Proto":"HTTP/1.1","ProtoMajor":1,"ProtoMinor":1,"Header":{"Accept-Encoding":["gzip"],"Content-Length":["16"],"Content-Type":["application/x-www-form-urlencoded"],"User-Agent":["Go-http-client/1.1"]},"ContentLength":16,"TransferEncoding":null,"Close":false,"Host":"127.0.0.1:55987","Form":null,"PostForm":null,"MultipartForm":null,"Trailer":null,"RemoteAddr":"127.0.0.1:55988","RequestURI":"/some/url?some_key=some_value\u0026some_other_key=some_other_value","TLS":null,"Body":"id=123\u0026key=Value","Cancel":null}

```