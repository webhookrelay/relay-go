package forward

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/urfave/negroni"
)

// ReceivedWebhook - keeps info about received webhook
type ReceivedWebhook struct {
	Payload    string
	ReceivedAt time.Time
	Method     string
	Headers    map[string]string
}

type WebhookServer struct {
	received []*ReceivedWebhook
	srv      *http.Server
	mw       MWFunc
}

type MWFunc func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc)

func (s *WebhookServer) SetMiddleware(mw MWFunc) {
	s.mw = mw
}

func (s *WebhookServer) Received() []*ReceivedWebhook {
	return s.received
}

func (s *WebhookServer) Cleanup() {
	s.received = []*ReceivedWebhook{}
}

func (s *WebhookServer) Start(port string) {

	mux := http.NewServeMux()

	mu := &sync.RWMutex{}

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("server", "webhook-server")
		w.WriteHeader(http.StatusOK)
	})

	// Incoming webhook handler
	mux.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		bd, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		defer r.Body.Close()

		// appending webhook to received list
		mu.Lock()

		headers := make(map[string]string)

		for k, vv := range r.Header {
			headers[k] = vv[0]
		}

		s.received = append(s.received, &ReceivedWebhook{
			ReceivedAt: time.Now(),
			Payload:    string(bd),
			Method:     r.Method,
			Headers:    headers,
		})
		mu.Unlock()

		w.Header().Add("server", "webhook-demo")

		fmt.Printf("webhook received, payload: %s, method: %s \n", string(bd), r.Method)
		w.WriteHeader(http.StatusOK)
	})

	n := negroni.New(negroni.NewRecovery())

	if s.mw != nil {
		n.Use(negroni.HandlerFunc(s.mw))
	}

	n.UseHandler(mux)

	s.srv = &http.Server{Addr: port, Handler: n}

	fmt.Printf("Receiving webhooks on http://localhost%s/webhook \n", port)
	// starting server
	if err := s.srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("listen: %s\n", err)
	}
}

func (s *WebhookServer) Shutdown() {
	if s.srv != nil {
		s.srv.Shutdown(context.Background())
	}
}

type FailureMiddleware struct {
	toFail     int
	statusCode int

	currentFails int
	mu           *sync.Mutex
}

func NewFailureMiddleware(toFail, statusCode int) *FailureMiddleware {
	return &FailureMiddleware{
		toFail:     toFail,
		statusCode: statusCode,
		mu:         &sync.Mutex{},
	}
}

func (mw *FailureMiddleware) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	mw.mu.Lock()
	defer mw.mu.Unlock()

	if mw.toFail == 0 {
		next(rw, r)
		return
	}

	if mw.currentFails < mw.toFail {
		rw.WriteHeader(mw.statusCode)
		mw.currentFails++
		return
	}

	next(rw, r)
}
