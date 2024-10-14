package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDisableCacheInDevMode(t *testing.T) {
	rr := httptest.NewRecorder()

	r, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	app := application{
		config: config{
			env: "development",
		},
	}

	app.disableCacheInDevMode(next).ServeHTTP(rr, r)

	rs := rr.Result()

	cacheControl := rs.Header.Get("Cache-Control")
	if cacheControl != "no-store" {
		t.Errorf("want %s, got %s", "no-store", cacheControl)
	}

}
