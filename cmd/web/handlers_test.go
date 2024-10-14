package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	internalWS "github.com/SegniAdebaGodsSon/internal/websocket"
)

func TestPing(t *testing.T) {
	app := newTestApplication(t)
	ts := newTestServer(t, app.routes())
	defer ts.Close()

	code, _, body := ts.get(t, "/ping")

	if code != http.StatusOK {
		t.Errorf("want %d; got %d", http.StatusOK, code)
	}

	if string(body) != "OK" {
		t.Errorf("want body to equal %q", "OK")
	}
}

func TestHome(t *testing.T) {
	app := newTestApplication(t)
	ts := newTestServer(t, app.routes())
	defer ts.Close()

	code, _, body := ts.get(t, "/Home")

	if code != http.StatusOK {
		t.Errorf("want %d; got %d", http.StatusOK, code)
	}

	if !strings.Contains(string(body), "<title>Home</title>") {
		t.Errorf("want body to contain %q", "<title>Home</title>")
	}
}

func TestAbout(t *testing.T) {
	app := newTestApplication(t)
	ts := newTestServer(t, app.routes())
	defer ts.Close()

	code, _, body := ts.get(t, "/about")

	if code != http.StatusOK {
		t.Errorf("want %d; got %d", http.StatusOK, code)
	}

	if !strings.Contains(string(body), "<title>About</title>") {
		t.Errorf("want body to contain %q", "<title>About</title>")
	}
}

func TestPostChat(t *testing.T) {
	t.Run("Incorrect path value", func(t *testing.T) {
		app := newTestApplication(t)
		ts := newTestServer(t, app.routes())
		defer ts.Close()

		code, _, body := ts.postForm(t, "/chat/wrong", url.Values{})

		if code != http.StatusBadRequest {
			t.Errorf("want %d; got %d", http.StatusBadRequest, code)
		}

		if !strings.Contains(string(body), "invalid chat type") {
			t.Errorf("want body to contain error: %q", "invalid chat type")
		}
	})

	invalidInterestTests := []struct {
		name           string
		path           string
		interests      url.Values
		wantBody       []byte
		wantStatusCode int
	}{
		{
			name: "Interests above 3",
			path: "/chat/text",
			interests: (func() url.Values {
				values := url.Values{}
				values.Add("interests[]", "sports")
				values.Add("interests[]", "film")
				values.Add("interests[]", "books")
				values.Add("interests[]", "games")
				values.Add("interests[]", "esports")
				return values
			})(),
			wantBody:       []byte("maximum of only 3 interests allowed"),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "Interest length too short",
			path: "/chat/text",
			interests: (func() url.Values {
				values := url.Values{}
				values.Add("interests[]", "")
				values.Add("interests[]", "film")
				values.Add("interests[]", "books")
				return values
			})(),
			wantBody:       []byte("interest too short (min 1 character)"),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "Interest length too long",
			path: "/chat/text",
			interests: (func() url.Values {
				values := url.Values{}
				values.Add("interests[]", "this interest is way too long to be valid")
				values.Add("interests[]", "film")
				values.Add("interests[]", "books")
				return values
			})(),
			wantBody:       []byte("interest too long (max 25 characters)"),
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range invalidInterestTests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication(t)
			ts := newTestServer(t, app.routes())
			defer ts.Close()
			code, _, body := ts.postForm(t, tt.path, tt.interests)

			if code != tt.wantStatusCode {
				t.Errorf("want %d; got %d", tt.wantStatusCode, code)
			}

			if !strings.Contains(string(body), string(tt.wantBody)) {
				t.Errorf("want body to contain error: %q", tt.wantBody)
			}
		})
	}

	// TODO: on successful submission of interests
	//	- check size of online users, has to be one
	//	- check the session "clientSessionId"
	//	- check the client's existence in OnlineClients
	//	- check status (status see other - 303)
	//	- check HTMX redirect header

	validInterestTests := []struct {
		name           string
		path           string
		interests      url.Values
		wantBody       []byte
		wantStatusCode int
		wantHeader     map[string]string
	}{
		{
			name: "Valid",
			path: "/chat/text",
			interests: (func() url.Values {
				values := url.Values{}
				values.Add("interests[]", "sports")
				values.Add("interests[]", "film")
				values.Add("interests[]", "books")
				return values
			})(),
			wantBody:       []byte(""),
			wantStatusCode: http.StatusSeeOther,
			wantHeader: map[string]string{
				"HX-Redirect": "/chat",
			},
		},
	}

	for _, tt := range validInterestTests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication(t)
			ts := newTestServer(t, app.routes())
			defer ts.Close()

			code, header, _ := ts.postForm(t, tt.path, tt.interests)

			// TEST: client must exist
			var client *internalWS.Client
			app.hub.OnlineClients.Range(func(_, value interface{}) bool {
				client = value.(*internalWS.Client)
				return false
			})

			if client == nil {
				t.Error("client was not registered")
			}

			// TEST: number of clients must be 1
			if size := app.hub.OnlineClients.Size(); size != 1 {
				t.Errorf("want the size of onlineClients to be 1, got=%d", size)
			}

			// TEST: cookies must be properly being set
			cookie := header["Set-Cookie"]

			if len(cookie) == 0 {
				t.Errorf("cookie not set")
				return
			}
			sessionCookie := cookie[0]

			req, err := http.NewRequest(http.MethodGet, ts.URL+"/chat", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Cookie", sessionCookie)

			rr := httptest.NewRecorder()
			app.session.Enable(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

				clientSessionID := app.session.GetString(r, "clientSessionId")

				if clientSessionID == "" {
					t.Errorf("session ID not found in session store")
				}

				_, exists := app.hub.OnlineClients.GetClient(clientSessionID)
				if !exists {
					t.Errorf("client with session ID='%s' not found in OnlineClients", clientSessionID)
				}
			})).ServeHTTP(rr, req)

			// TEST: test interests matching
			if client != nil {
				for _, ti := range tt.interests["interests[]"] {
					found := false
					for _, ci := range client.Interests {
						if ti == ci {
							found = true
							break
						}
					}

					if !found {
						t.Errorf("want interest='%s' to exist", ti)
					}
				}

			}

			// TEST: test status code
			if code != tt.wantStatusCode {
				t.Errorf("want %d; got %d", tt.wantStatusCode, code)
			}

			// TEST: test headers
			for key, value := range tt.wantHeader {
				if header.Get(key) != value {
					t.Errorf("want header value for %s to be %s; got %s", key, value, header.Get(key))
				}
			}
		})
	}
}
