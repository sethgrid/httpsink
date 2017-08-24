package httpsink

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

var Logger = log.New(os.Stderr, "[httpsink] ", log.LstdFlags)

// Server provides the struct around running the HTTP sync
type Server struct {
	*http.Server
	Capacity int
	Proxy    string
	TTL      time.Duration

	mutex    sync.RWMutex
	requests []Request
	timer    *time.Timer
}

// New creates a sync running on :0 (random port)
func New() *Server {
	return &Server{
		Server:   &http.Server{},
		requests: make([]Request, 0),
	}
}

func (s *Server) ListenAndServe() error {
	s.initializeRouter()

	return s.Server.ListenAndServe()
}

func (s *Server) ListenAndServeTLS(certFile, keyFile string) error {
	s.initializeRouter()

	return s.Server.ListenAndServeTLS(certFile, keyFile)
}

func (s *Server) setHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Content-Type", "application/json")

	defer s.mutex.Unlock()

	s.mutex.Lock()
	count := len(s.requests)

	if s.Capacity != 0 && count >= s.Capacity {
		Logger.Println("sink at capacity")

		http.Error(rw, `{"errors":[{"message": "http sink is at capacity"}]}`, http.StatusInsufficientStorage)
		return
	}

	buf := bytes.NewBuffer(nil)
	if r.Body != nil {
		body := r.Body
		io.Copy(buf, body)
		r.Body = ioutil.NopCloser(buf)

		defer body.Close()
	}

	req := Request{
		URL:        r.URL.String(),
		Method:     r.Method,
		Body:       buf.Bytes(),
		Headers:    map[string][]string(r.Header),
		timestamp:  time.Now(),
		recipients: mailRecipients(r),
	}

	Logger.Printf("storing request %d", len(s.requests)+1)
	s.requests = append(s.requests, req)

	if s.Proxy != "" {
		go s.proxy(s.Proxy, r)
	}

	rw.Write([]byte(`{"message":"success"}`))
}

func (s *Server) proxy(proxy string, r *http.Request) {
	path := r.URL.Path
	if r.URL.RawQuery != "" {
		path = fmt.Sprintf("%s?%s", path, r.URL.RawQuery)
	}

	url := proxy + path

	buf := bytes.NewBuffer(nil)
	io.Copy(buf, r.Body)
	defer r.Body.Close()

	outgoing, _ := http.NewRequest(r.Method, url, buf)

	for k, v := range r.Header {
		outgoing.Header[k] = v
	}

	res, err := http.DefaultClient.Do(outgoing)
	if err != nil {
		Logger.Printf("proxy error: %s", err)
		return
	}

	Logger.Printf("proxy response status code: %d", res.StatusCode)
}

func (s *Server) getHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Content-Type", "application/json")

	// index starts at 1
	index, err := strconv.Atoi(mux.Vars(r)["index"])
	log.Printf("getting index %d", index)

	if err != nil {
		http.Error(rw, `{"errors":[{"message":"index is not an integer"}]}`, http.StatusBadRequest)
		return
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if index > len(s.requests) {
		http.Error(rw, `{"errors":[{"message":"index is out of range"}]}`, http.StatusBadRequest)
		return
	}

	json.NewEncoder(rw).Encode(s.requests[index-1])
}

func (s *Server) lastHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Content-Type", "application/json")

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	index := len(s.requests)
	if index == 0 {
		http.Error(rw, `{"errors":[{"message":"no requests have been received"}]}`, http.StatusNotFound)
	}

	if index > len(s.requests) {
		http.Error(rw, `{"errors":[{"message":"index is out of range"}]}`, http.StatusBadRequest)
		return
	}

	log.Printf("getting last index %d", index)

	json.NewEncoder(rw).Encode(s.requests[index-1])
}

func (s *Server) recipientHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Content-Type", "application/json")

	email := mux.Vars(r)["recipient"]
	if email == "" {
		http.Error(rw, `{"errors":[{"message":"missing recipient"}]}`, http.StatusBadRequest)
		return
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	requests := make([]Request, 0)

	for _, req := range s.requests {
		for _, recipient := range req.recipients {
			if strings.EqualFold(email, recipient) {
				requests = append(requests, req)
				break
			}
		}
	}

	type RecipeintResponse struct {
		Requests []Request `json:"requests"`
	}

	json.NewEncoder(rw).Encode(RecipeintResponse{Requests: requests})
}

func (s *Server) clearHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Content-Type", "application/json")

	s.mutex.Lock()
	defer s.mutex.Unlock()

	Logger.Println("clearing requests")
	s.requests = make([]Request, 0)

	rw.WriteHeader(http.StatusNoContent)
}

func (s *Server) healthcheck(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Content-Type", "application/json")
	rw.WriteHeader(http.StatusNoContent)
}

func (s *Server) initializeRouter() {
	router := mux.NewRouter()
	router.HandleFunc("/requests", s.clearHandler).Methods("DELETE")
	router.HandleFunc("/requests", s.getHandler).Methods("GET")
	router.HandleFunc("/requests/last", s.lastHandler).Methods("GET")
	router.HandleFunc("/requests/recipient/{recipient}", s.recipientHandler).Methods("GET")
	router.HandleFunc("/request/{index}", s.getHandler).Methods("GET")
	router.HandleFunc("/healthcheck", s.healthcheck).Methods("GET")

	// any other routes get handled here
	router.NotFoundHandler = http.HandlerFunc(s.setHandler)

	s.Server.Handler = router

	s.timer = time.AfterFunc(time.Second, s.clearOldRequests)
}

func (s *Server) clearOldRequests() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.TTL > 0 {
		now := time.Now()

		for i := len(s.requests); i >= 0; i-- {
			d := s.requests[i].timestamp.Sub(now)
			if d > s.TTL {
				s.requests = append(s.requests[:i], s.requests[i+1:]...)
			}
		}
	}
}

type Request struct {
	URL     string              `json:"url"`
	Method  string              `json:"method"`
	Body    []byte              `json:"body"`
	Headers map[string][]string `json:"headers"`

	recipients []string
	timestamp  time.Time
}

func mailRecipients(r *http.Request) []string {
	type XSMTPAPIHeader struct {
		To               []string `json:"to"` // v2 header
		Personalizations []struct {
			To []struct {
				Email string `json:"email"`
			} `json:"to"`
		} `json:"personalizations"` // v3 header
	}

	recipients := make([]string, 0)

	to := r.URL.Query().Get("to") // v2 to URL param
	if to != "" {
		recipients = append(recipients, to)
	}

	tos, ok := r.URL.Query()["to[]"]
	if ok {
		recipients = append(recipients, tos...)
	}

	header := r.Header.Get("x-smtpapi")
	if header != "" {
		data := XSMTPAPIHeader{}
		err := json.Unmarshal([]byte(header), &data)
		if err != nil {
			Logger.Printf("invalid x-smptapi header: %s", err)
			return recipients
		}

		if data.To != nil {
			recipients = append(recipients, data.To...)
		}

		for _, p := range data.Personalizations {
			for _, to := range p.To {
				recipients = append(recipients, to.Email)
			}
		}
	}

	return recipients
}
