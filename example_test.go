package httpsink_test

import "github.com/sendgrid/httpsink"

func Example() {
	sink := httpsink.New()
	sink.Addr = "0.0.0.0:1234"

	sink.ListenAndServe()
}
