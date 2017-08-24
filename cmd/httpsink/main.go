package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/sendgrid/httpsink"
)

func main() {
	var (
		port     int
		host     string
		proxy    string
		ttl      time.Duration
		capacity int
	)

	flag.IntVar(&port, "port", 50111, "port to run on")
	flag.StringVar(&host, "host", "0.0.0.0", "network interface to run on")
	flag.IntVar(&capacity, "capacity", 0, "sink capacity")
	flag.StringVar(&proxy, "proxy", "", "URL to proxy to")
	flag.DurationVar(&ttl, "ttl", 0, "TTL of sink requests")

	flag.Parse()

	sink := httpsink.New()
	sink.Addr = fmt.Sprintf("%s:%d", host, port)
	sink.Proxy = proxy
	sink.Capacity = capacity
	sink.TTL = ttl

	log.Printf("listening on %s", sink.Server.Addr)

	err := sink.ListenAndServe()
	if err != nil {
		log.Fatalf("error listening: %s", err)
	}
}
