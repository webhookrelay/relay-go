package forward

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"net/http/httptest"
	"testing"

	"github.com/webhookrelay/relay-go/pkg/types"
)

func TestRelaySuccess(t *testing.T) {
	relayed := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello from the other side")
		relayed = true
	}))
	defer ts.Close()

	dr := NewDefaultForwarder(&Opts{Retries: 1})

	wr := types.Event{
		Meta: types.EventMeta{
			OutputDestination: ts.URL,
		},
		Method: http.MethodGet,
		Body:   "hi",
	}

	ws := dr.Forward(wr)
	if ws.StatusCode != 200 {
		t.Errorf("unexpected status: %d", ws.StatusCode)
	}

	if !relayed {
		t.Errorf("failed to relay")
	}
}

func TestRelayCheckBody(t *testing.T) {
	payload := "important payload"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bts, _ := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if string(bts) != string(payload) {
			t.Errorf("payloads do not match")
		}
	}))
	defer ts.Close()

	dr := NewDefaultForwarder(&Opts{Retries: 1})
	wr := types.Event{
		Meta: types.EventMeta{
			OutputDestination: ts.URL,
		},
		Method: http.MethodGet,
		Body:   payload,
	}

	ws := dr.Forward(wr)
	if ws.StatusCode != 200 {
		t.Errorf("failed to relay, got status: %d", ws.StatusCode)
	}
}

func TestRelayRetryOnce(t *testing.T) {
	payload := "important payload"

	failed := false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !failed {
			w.WriteHeader(http.StatusInternalServerError)
			failed = true
			return
		}

		w.WriteHeader(http.StatusOK)
		return
	}))
	defer ts.Close()

	dr := NewDefaultForwarder(&Opts{Retries: 1})

	wr := types.Event{
		Meta: types.EventMeta{
			OutputDestination: ts.URL,
		},
		Method: http.MethodGet,
		Body:   payload,
	}

	ws := dr.Forward(wr)
	if ws.StatusCode != 200 {
		t.Errorf("failed to relay, got status: %d", ws.StatusCode)
	}
	if ws.Retries != 1 {
		t.Errorf("expected 1 retry, got: %d", ws.Retries)
	}
}

func TestRelayGiveUp(t *testing.T) {
	payload := "important payload"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("fail"))
	}))
	defer ts.Close()

	dr := NewDefaultForwarder(&Opts{Retries: 1})

	wr := types.Event{
		Meta: types.EventMeta{
			OutputDestination: ts.URL,
		},
		Method: http.MethodGet,
		Body:   payload,
	}

	ws := dr.Forward(wr)
	if ws.StatusCode != http.StatusInternalServerError {
		t.Errorf("should have failed with 500, got: %d", ws.StatusCode)
	}

	if ws.Retries != 1 {
		t.Errorf("unexpected amount of retries: %d", ws.Retries)
	}
}

func TestRelayRetryTwice(t *testing.T) {
	payload := "important payload"

	failMax := 2
	fail := 0

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail <= failMax {
			w.WriteHeader(http.StatusInternalServerError)
			fail++
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	dr := NewDefaultForwarder(&Opts{Retries: 3})

	wr := types.Event{
		Meta: types.EventMeta{
			OutputDestination: ts.URL,
		},
		Method: http.MethodGet,
		Body:   payload,
	}

	ws := dr.Forward(wr)
	if ws.StatusCode != http.StatusOK {
		t.Errorf("should have failed with 200, got: %d", ws.StatusCode)
	}

	if ws.Retries != 3 {
		t.Errorf("unexpected amount of retries: %d", ws.Retries)
	}
}
