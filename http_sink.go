package httpsink

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/gorilla/mux"
)

var Logger = log.New(os.Stderr, "[httpsink] ", log.LstdFlags)

// Server provides the struct around running the HTTP sync
type Server struct {
	*http.Server
	Capacity int

	mutex    sync.RWMutex
	requests []Request
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
		io.Copy(buf, r.Body)
		defer r.Body.Close()
	}

	req := Request{
		URL:     r.URL.String(),
		Method:  r.Method,
		Body:    buf.Bytes(),
		Headers: map[string][]string(r.Header),
	}

	Logger.Printf("storing request %d", len(s.requests)+1)
	s.requests = append(s.requests, req)

	rw.WriteHeader(http.StatusNoContent)
}

func (s *Server) getHandler(rw http.ResponseWriter, r *http.Request) {
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

func (s *Server) clearHandler(rw http.ResponseWriter, r *http.Request) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	Logger.Println("clearing requests")
	s.requests = make([]Request, 0)

	rw.WriteHeader(http.StatusNoContent)
}

func (s *Server) healthcheck(rw http.ResponseWriter, r *http.Request) {
	rw.WriteHeader(http.StatusNoContent)
}

func (s *Server) initializeRouter() {
	router := mux.NewRouter()
	router.HandleFunc("/", s.setHandler)
	router.HandleFunc("/requests", s.clearHandler).Methods("DELETE")
	router.HandleFunc("/requests", s.getHandler).Methods("GET")
	router.HandleFunc("/requests/last", s.lastHandler).Methods("GET")
	router.HandleFunc("/request/{index}", s.getHandler).Methods("GET")
	router.HandleFunc("/healthcheck", s.healthcheck).Methods("GET")

	s.Server.Handler = router
}

type Request struct {
	URL     string              `json:"url"`
	Method  string              `json:"method"`
	Body    []byte              `json:"body"`
	Headers map[string][]string `json:"headers"`
}
