package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRestAPIWriterOpen(t *testing.T) {
	testCases := []struct {
		name     string
		rw       restAPIWriter
		setup    func() error
		wantErr  bool
		errValue string
	}{
		{
			name: "When client already exists, does nothing and returns nil",
			rw:   restAPIWriter{client: &http.Client{}},
		},
		{
			name:     "When URL is broken, returns error",
			rw:       restAPIWriter{URL: "\x1F"},
			wantErr:  true,
			errValue: `restAPIWriter: Error parsing URL: parse "\x1f": net/url: invalid control character in URL`,
		},
		{
			name: "When URL is http, creates client with no TLS and returns nil",
			rw:   restAPIWriter{URL: "http://localhost"},
		},
		{
			name: "When URL is https and has no TLS, creates client with basic TLS and returns nil",
			rw:   restAPIWriter{URL: "https://localhost"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				err := tc.setup()
				if err != nil {
					t.Fatal(err)
				}
			}

			err := tc.rw.open()
			if tc.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tc.errValue, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}

}

func TestRestAPIWriter_Close(t *testing.T) {
	writer := &restAPIWriter{
		client: &http.Client{},
	}

	t.Run("Closes the writer successfully", func(t *testing.T) {
		writer.close()
		assert.Nil(t, writer.client)
	})
}

func TestRestAPIWriter_Write(t *testing.T) {
	testCases := []struct {
		name     string
		setup    func() (*restAPIWriter, *httptest.Server)
		wantErr  bool
		errValue string
	}{
		{
			name: "returns error when client is nil",
			setup: func() (*restAPIWriter, *httptest.Server) {
				writer := &restAPIWriter{}
				writer.client = nil
				return writer, nil
			},
			wantErr:  true,
			errValue: "restAPIWriter: client is nil",
		},
		{
			name: "returns error when NewRequest fails",
			setup: func() (*restAPIWriter, *httptest.Server) {
				writer := &restAPIWriter{
					client: &http.Client{},
					URL:    string([]byte{0x7f}), // invalid URL
				}
				return writer, nil
			},
			wantErr:  true,
			errValue: "restAPIWriter: Error creating request",
		},
		{
			name: "returns error when client.Do fails",
			setup: func() (*restAPIWriter, *httptest.Server) {
				writer := &restAPIWriter{
					client: &http.Client{},
					URL:    "http://localhost",
					Port:   12345, // no server listening on this port
				}
				return writer, nil
			},
			wantErr:  true,
			errValue: "restAPIWriter: Error sending request",
		},
		{
			name: "returns error when response status code is >= 300",
			setup: func() (*restAPIWriter, *httptest.Server) {
				testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusBadGateway)
				}))

				// gets host and port from test server
				parsedUrl, err := url.Parse(testServer.URL)
				if err != nil {
					t.Fatal(err)
				}

				host, portStr, err := net.SplitHostPort(parsedUrl.Host)
				if err != nil {
					t.Fatal(err)
				}

				port, err := strconv.Atoi(portStr)
				if err != nil {
					t.Fatal(err)
				}

				writer := &restAPIWriter{
					client: &http.Client{},
					URL:    "http://" + host,
					Port:   port,
				}
				return writer, testServer
			},
			wantErr:  true,
			errValue: "restAPIWriter: Server response: 502 Bad Gateway",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			writer, server := tc.setup()
			if server != nil {
				defer server.Close()
			}

			err := writer.write([]byte("test"))
			if tc.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errValue)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
