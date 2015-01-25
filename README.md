HTTPSink
======================

Caputure all requests that come into the server and be able to get those requests later for examination. Useful for faking API endpoints.

Using configuration or environment variables, you can set the foreign API endpoint in your code to use "localhost:8888" (or any port). Now, in tests, you can use httpsink to stand up an HTTP sink that will either just swallow up your requests or you can choose for it to give desired responses.

Example:

```go
func TestSomeCode(t *testing.T) {
  // set up the sink (optionally set the port with the NewHTTPSinkOnAdder("localhost:8888"))
  hSync, _ := NewHTTPSink()
	defer hSync.Close()

	expectedBody := []byte(`{"key":"value"}`)
	hSync.SetNextResponse(&SimpleResponseWriter{StatusCode: http.StatusTeapot, Body: expectedBody})
	
	// make call to your code that in-turn makes a call to an api
	err := sendAPIRequest()
	if err != nil{
	  t.Errorf("error sending api request - %s", err)
	}
	
	// optionaly verify that the foreign api got the right request if you like
	getURL := fmt.Sprintf("http://%s/get?idx=0", hSync.Addr)
	getResp, _ := http.Get(getURL)

	capturedRequest := http.Request{}

	_ = json.NewDecoder(getResp.Body).Decode(&capturedRequest)

	if !strings.Contains(capturedRequest.URL.RawQuery, "some_key=some_value") {
		t.Errorf("captured request in sync did not capture proper query - %s", capturedRequest.URL.RawQuery)
	}
```
