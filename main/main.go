package main

import "net/http"
import "fmt"

import "github.com/sethgrid/httpsink"

func main() {
    // set up the sink (optionally set the port with the NewHTTPSinkOnAdder("localhost:8888"))
    hSync, err := httpsink.NewHTTPSinkOnAddr("localhost:8899", 0)
    if err != nil {
      fmt.Println(err.Error())
    }
    defer hSync.Close()

    // set the behavior of the foreign API endpoint
    expectedBody := []byte(`{"key":"value"}`)
    hSync.SetResponse(&httpsink.SimpleResponseWriter{StatusCode: http.StatusOK, Body: expectedBody})

    hSync.StartHTTP()

    // // make call to your code that in-turn makes a call to the foreign API endpoint
    // err := sendAPIRequest()
    // if err != nil{
    //   t.Errorf("error sending api request - %s", err)
    // }

    // // optionaly verify that the foreign API got the right request if you like
    // getURL := fmt.Sprintf("http://%s/get?request_number=0", hSync.Addr)
    // getResp, _ := http.Get(getURL)
    // defer getResp.Body.Close()

    // capturedRequest := http.Request{}

    // ignoring the error because http.Request.Body does not play well with json.Decode
    // it works for this example, but you could also make a custom struct or use simplejson
    // if using a custom struct, just embedd *http.Request and mask with a Body interface{} property
    // _ = json.NewDecoder(getResp.Body).Decode(&capturedRequest)

    // if !strings.Contains(capturedRequest.URL.RawQuery, "some_key=some_value") {
    //     t.Errorf("captured request in sync did not capture proper query - %s", capturedRequest.URL.RawQuery)
    // }
  }
