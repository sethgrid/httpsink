package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/sendgrid/httpsink"
)

func main() {
	port := 50111
	strPort := os.Getenv("HTTPSINK_PORT")
	if strPort != "" {
		var err error
		port, err = strconv.Atoi(strPort)
		if err != nil {
			log.Fatalf("unable to parse HTTPSINK_PORT: %s", err)
		}
	}

	iface := os.Getenv("HTTPSINK_INTERFACE")
	if iface == "" {
		iface = "0.0.0.0"
	}

	sink := httpsink.New()
	sink.Server.Addr = fmt.Sprintf("%s:%d", iface, port)

	log.Printf("listening on %s", sink.Server.Addr)

	err := sink.ListenAndServe()
	if err != nil {
		log.Fatalf("error listening: %s", err)
	}
}
