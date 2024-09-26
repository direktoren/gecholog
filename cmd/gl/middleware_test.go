package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/direktoren/gecholog/internal/gechologobject"
	"github.com/direktoren/gecholog/internal/protectedheader"
	"github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
)

func Test_loggingPostProcessorFunc_BadArgs(t *testing.T) {
	type args struct {
		nc                *nats.Conn
		subject           string
		lastError         func(error, time.Time)
		lastTransactionID func(string)
		postHandler       http.Handler
		logFilter         finalLogger
		s                 *state
	}
	tests := []struct {
		name        string
		args        args
		expextedNil bool
	}{
		{
			name: "basic success",
			args: args{
				nc:                &nats.Conn{},
				subject:           "subject",
				lastError:         func(error, time.Time) {},
				lastTransactionID: func(string) {},
				postHandler:       http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
				logFilter:         finalLogger{},
				s: &state{
					m: &sync.Mutex{},
				},
			},
			expextedNil: false,
		},
		{
			name: "nc is nil",
			args: args{
				nc:                nil, // This is the error
				subject:           "subject",
				lastError:         func(error, time.Time) {},
				lastTransactionID: func(string) {},
				postHandler:       http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
				logFilter:         finalLogger{},
				s: &state{
					m: &sync.Mutex{},
				},
			},
			expextedNil: true,
		},
		{
			name: "subject is empty",
			args: args{
				nc:                &nats.Conn{},
				subject:           "", // This is the error
				lastError:         func(error, time.Time) {},
				lastTransactionID: func(string) {},
				postHandler:       http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
				logFilter:         finalLogger{},
				s: &state{
					m: &sync.Mutex{},
				},
			},
			expextedNil: true,
		},
		{
			name: "lastError is nil",
			args: args{
				nc:                &nats.Conn{},
				subject:           "subject",
				lastError:         nil, // This is the error
				lastTransactionID: func(string) {},
				postHandler:       http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
				logFilter:         finalLogger{},
				s: &state{
					m: &sync.Mutex{},
				},
			},
			expextedNil: true,
		},
		{
			name: "lastTransactionID is nil",
			args: args{
				nc:                &nats.Conn{},
				subject:           "subject",
				lastError:         func(error, time.Time) {},
				lastTransactionID: nil, // This is the error
				postHandler:       http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
				logFilter:         finalLogger{},
				s: &state{
					m: &sync.Mutex{},
				},
			},
			expextedNil: true,
		},
		{
			name: "s is nil",
			args: args{
				nc:                &nats.Conn{},
				subject:           "subject",
				lastError:         func(error, time.Time) {},
				lastTransactionID: func(string) {},
				postHandler:       http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
				logFilter:         finalLogger{},
				s:                 nil, // This is the error
			},
			expextedNil: true,
		},
		{
			name: "s mutex is nil",
			args: args{
				nc:                &nats.Conn{},
				subject:           "subject",
				lastError:         func(error, time.Time) {},
				lastTransactionID: func(string) {},
				postHandler:       http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
				logFilter:         finalLogger{},
				s:                 &state{ // This is the error
				},
			},
			expextedNil: true,
		},
		{
			name: "postHandler is nil",
			args: args{
				nc:                &nats.Conn{},
				subject:           "subject",
				lastError:         func(error, time.Time) {},
				lastTransactionID: func(string) {},
				postHandler:       nil, // This is the error
				logFilter:         finalLogger{},
				s: &state{
					m: &sync.Mutex{},
				},
			},
			expextedNil: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := loggingPostProcessorFunc(tt.args.nc, tt.args.subject, tt.args.lastError, tt.args.lastTransactionID, tt.args.postHandler, tt.args.logFilter, tt.args.s)
			if tt.expextedNil {
				assert.Nil(t, f)
				return
			}
			assert.NotNil(t, f)

		})
	}
}

func Test_loggingPostProcessorFunc(t *testing.T) {

	opts := test.DefaultTestOptions
	opts.Port = -1 // Random port
	server := test.RunServer(&opts)
	defer server.Shutdown()

	// Connect to the in-memory NATS server
	nc, err := nats.Connect(server.ClientURL())
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()

	subject := "testsubject"
	// Subscribe and check message
	sub, err := nc.SubscribeSync(subject)
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Unsubscribe()

	var lastError error
	lastErrorFunction := func(err error, t time.Time) {
		lastError = err
	}
	var lastTransactionID string
	lastTransactionIDFunction := func(transactionID string) {
		lastTransactionID = transactionID
	}

	s := &state{
		m: &sync.Mutex{},
	}

	type args struct {
		requestResponseProcessorHandler http.HandlerFunc
		finalLogger                     finalLogger
	}
	tests := []struct {
		name                  string
		args                  args
		expextedStatusCode    int
		expectedTransactionID string
		expectedErrorStr      string
		expectedLog           map[string]json.RawMessage
	}{
		{
			name: "basic success",
			args: args{
				requestResponseProcessorHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					crw, ok := w.(*GechologResponseWriter)
					if !ok {
						http.Error(w, "Internal Server Error", http.StatusInternalServerError)
						return
					}
					crw.transactionID = "GATEWAYID_1696681410696216000_1_0"
					crw.WriteHeader(http.StatusOK)
				}),
				finalLogger: finalLogger{},
			},
			expextedStatusCode:    http.StatusOK,
			expectedTransactionID: "GATEWAYID_1696681410696216000_1_0",
			expectedErrorStr:      "",
			expectedLog: map[string]json.RawMessage{
				"transaction_id": json.RawMessage(`"GATEWAYID_1696681410696216000_1_0"`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			s.countRequests = 0 // reset the counter
			postProcessor := loggingPostProcessorFunc(nc, subject, lastErrorFunction, lastTransactionIDFunction, tt.args.requestResponseProcessorHandler, tt.args.finalLogger, s)
			assert.NotNil(t, postProcessor)

			// Create a ResponseRecorder to record the response
			rr := httptest.NewRecorder()
			g := &GechologResponseWriter{
				ResponseWriter: rr,
				outboundBody:   bytes.NewBufferString(""),
				inboundBody:    bytes.NewBufferString(""),
				egressBody:     bytes.NewBufferString(""),

				processorLogsRequestSync:   map[string]processorLog{},
				processorLogsResponseSync:  map[string]processorLog{},
				processorLogsRequestAsync:  map[string]processorLog{},
				processorLogsResponseAsync: map[string]processorLog{},

				requestObject:      gechologobject.New(),
				requestErrorObject: gechologobject.New(),

				responseObject:      gechologobject.New(),
				responseErrorObject: gechologobject.New(),

				rootObject:      gechologobject.New(),
				rootErrorObject: gechologobject.New(),
			}

			postProcessor(g) // Run it!

			assert.Equal(t, tt.expextedStatusCode, rr.Code)
			assert.Equal(t, tt.expectedTransactionID, lastTransactionID)

			receivedMsg, err := sub.NextMsg(nats.DefaultTimeout)
			if err != nil {
				t.Fatal(err)
			}

			rootObject := gechologobject.New()
			err = json.Unmarshal(receivedMsg.Data, &rootObject)
			assert.NoError(t, err)

			for k, v := range tt.expectedLog {
				response, err := rootObject.GetField(k)
				if err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, v, response)
			}

			if tt.expectedErrorStr == "" {
				assert.Nil(t, lastError)
				return
			}
			assert.Equal(t, tt.expectedErrorStr, lastError.Error())
		})
	}

}

func Test_loggingMiddlewareFunc_BadArgs(t *testing.T) {
	type args struct {
		processor loggingPostProcessor
	}
	tests := []struct {
		name        string
		args        args
		expextedNil bool
	}{
		{
			name: "basic success",
			args: args{
				processor: func(w *GechologResponseWriter) {},
			},
			expextedNil: false,
		},
		{
			name: "loggingPostProcessor is nil",
			args: args{
				processor: nil, // This is the error
			},
			expextedNil: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := loggingMiddlewareFunc(tt.args.processor, false)
			if tt.expextedNil {
				assert.Nil(t, f)
				return
			}
			assert.NotNil(t, f)

		})
	}
}

func Test_loggingMiddlewareFunc(t *testing.T) {

	ch := make(chan int64)
	processor := func(w *GechologResponseWriter) {
		ch <- w.ingressEgressTimer.GetDuration()
	}
	middleware := loggingMiddlewareFunc(processor, false)
	if middleware == nil {
		t.Fatal("middleware is nil")
	}

	handlerToTest := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	if handlerToTest == nil {
		t.Fatal("handlerToTest is nil")
	}

	req, err := http.NewRequest("POST", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Create a handler and pass the request and recorder
	handler := http.Handler(handlerToTest)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	select {
	case d := <-ch:
		assert.GreaterOrEqual(t, d, int64(50))
	case <-time.After(1 * time.Second):
		t.Fatal("timeout")

	}

}

func Test_sessionMiddlewareFunc_BadArgs(t *testing.T) {
	type args struct {
		gatewayID      string
		sesssionHeader string
		s              *state
	}
	tests := []struct {
		name        string
		args        args
		expextedNil bool
	}{
		{
			name: "basic success",
			args: args{
				gatewayID:      "GATEWAYID",
				sesssionHeader: "Session-Header",
				s: &state{
					m: &sync.Mutex{},
				},
			},
			expextedNil: false,
		},
		{
			name: "gatewayID is empty",
			args: args{
				gatewayID:      "", // This is the error
				sesssionHeader: "Session-Header",
				s: &state{
					m: &sync.Mutex{},
				},
			},
			expextedNil: true,
		},
		{
			name: "bad gatewayID",
			args: args{
				gatewayID:      "gatewayID", // This is the error
				sesssionHeader: "Session-Header",
				s: &state{
					m: &sync.Mutex{},
				},
			},
			expextedNil: true,
		},
		{
			name: "sessionHeader is empty",
			args: args{
				gatewayID:      "GATEWAYID",
				sesssionHeader: "", // This is the error
				s: &state{
					m: &sync.Mutex{},
				},
			},
			expextedNil: true,
		},
		{
			name: "state is nil",
			args: args{
				gatewayID:      "GATEWAYID",
				sesssionHeader: "Session-Header",
				s:              nil, // This is the error
			},
			expextedNil: true,
		},
		{
			name: "s mutex is nil",
			args: args{
				gatewayID:      "GATEWAYID",
				sesssionHeader: "Session-Header",
				s: &state{
					m: nil, // This is the error
				},
			},
			expextedNil: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := sessionMiddlewareFunc(tt.args.gatewayID, tt.args.sesssionHeader, tt.args.s)
			if tt.expextedNil {
				assert.Nil(t, f)
				return
			}
			assert.NotNil(t, f)

		})
	}
}

func Test_sessionMiddlewareFunc_GechologResponseWriter(t *testing.T) {

	middleware := sessionMiddlewareFunc("GATEWAYID", "sessionHeader", &state{
		m: &sync.Mutex{},
	})
	assert.NotNil(t, middleware)

	handlerToTest := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	assert.NotNil(t, handlerToTest)

	type args struct {
		responseWriterToUse func(*httptest.ResponseRecorder) http.ResponseWriter
	}
	tests := []struct {
		name               string
		args               args
		expextedStatusCode int
	}{
		{
			name: "basic success",
			args: args{
				responseWriterToUse: func(rr *httptest.ResponseRecorder) http.ResponseWriter {
					g := &GechologResponseWriter{
						ResponseWriter: rr,
						inboundBody:    bytes.NewBufferString(""),
					}
					g.ingressEgressTimer.Start()
					return g
				},
			},
			expextedStatusCode: http.StatusOK,
		},
		{
			name: "not a GechologResponseWriter",
			args: args{
				responseWriterToUse: func(rr *httptest.ResponseRecorder) http.ResponseWriter {
					return http.ResponseWriter(rr)
				},
			},
			expextedStatusCode: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			req, err := http.NewRequest("POST", "/", nil)
			if err != nil {
				t.Fatal(err)
			}

			// Create a ResponseRecorder to record the response
			rr := httptest.NewRecorder()

			// Create a handler and pass the request and recorder
			handler := http.Handler(handlerToTest)
			w := tt.args.responseWriterToUse(rr)
			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expextedStatusCode, rr.Code)

		})
	}

}

func Test_sessionMiddlewareFunc_SessionID(t *testing.T) {

	s := &state{
		m: &sync.Mutex{},
	}
	middleware := sessionMiddlewareFunc("GATEWAYID", "Session-Header", s)
	assert.NotNil(t, middleware)

	handlerToTest := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	assert.NotNil(t, handlerToTest)

	type args struct {
		unixNanoStr string
		headers     http.Header
	}
	tests := []struct {
		name                  string
		args                  args
		expextedStatusCode    int
		expectedSessionID     string
		expectedTransactionID string
	}{
		{
			name: "basic success",
			args: args{
				unixNanoStr: "1696681410696216000",
				headers:     http.Header{},
			},
			expextedStatusCode:    http.StatusOK,
			expectedSessionID:     "GATEWAYID_1696681410696216000_1_0",
			expectedTransactionID: "GATEWAYID_1696681410696216000_1_0",
		},
		{
			name: "sessionID in header",
			args: args{
				unixNanoStr: "1696681410696216000",
				headers: http.Header{
					"Session-Header": []string{"GATEWAYID_4444681410696216000_1_2"},
				},
			},
			expextedStatusCode:    http.StatusOK,
			expectedSessionID:     "GATEWAYID_4444681410696216000_1_0",
			expectedTransactionID: "GATEWAYID_4444681410696216000_1_3",
		},
		{
			name: "sessionID wrong place in header",
			args: args{
				unixNanoStr: "1696681410696216000",
				headers: http.Header{
					"Session-Header": []string{"", "GATEWAYID_4444681410696216000_1_2"}, // Ignore this session id since on second position
				},
			},
			expextedStatusCode:    http.StatusOK,
			expectedSessionID:     "GATEWAYID_1696681410696216000_1_0",
			expectedTransactionID: "GATEWAYID_1696681410696216000_1_0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			s.countRequests = 0 // reset the counter

			// Convert the string to int64
			unixNano, err := strconv.ParseInt(tt.args.unixNanoStr, 10, 64)
			if err != nil {
				t.Fatal(err)
			}

			// Create a ResponseRecorder to record the response
			rr := httptest.NewRecorder()

			g := &GechologResponseWriter{
				ResponseWriter: rr,
				inboundBody:    bytes.NewBufferString(""),
			}
			g.ingressEgressTimer.SetStart(time.Unix(0, unixNano))

			req, err := http.NewRequest("POST", "/", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header = tt.args.headers

			// Create a handler and pass the request and recorder
			handler := http.Handler(handlerToTest)
			handler.ServeHTTP(g, req)

			assert.Equal(t, tt.expextedStatusCode, rr.Code)
			assert.Equal(t, tt.expectedSessionID, g.sessionID)
			assert.Equal(t, tt.expectedTransactionID, g.transactionID)

		})
	}

}

func Test_egressResponseMiddleware_GechologResponseWriter(t *testing.T) {

	handlerToTest := egressResponseMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	assert.NotNil(t, handlerToTest)

	type args struct {
		responseWriterToUse func(*httptest.ResponseRecorder) http.ResponseWriter
	}
	tests := []struct {
		name               string
		args               args
		expextedStatusCode int
	}{
		{
			name: "basic success",
			args: args{
				responseWriterToUse: func(rr *httptest.ResponseRecorder) http.ResponseWriter {
					g := &GechologResponseWriter{
						ResponseWriter: rr,
						egressHeaders:  http.Header{},
						egressBody:     bytes.NewBufferString("test"),
						inboundBody:    bytes.NewBufferString(""),
					}
					g.ingressEgressTimer.Start()
					return g
				},
			},
			expextedStatusCode: http.StatusOK,
		},
		{
			name: "not a GechologResponseWriter",
			args: args{
				responseWriterToUse: func(rr *httptest.ResponseRecorder) http.ResponseWriter {
					return http.ResponseWriter(rr)
				},
			},
			expextedStatusCode: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			req, err := http.NewRequest("POST", "/", nil)
			if err != nil {
				t.Fatal(err)
			}

			// Create a ResponseRecorder to record the response
			rr := httptest.NewRecorder()

			// Create a handler and pass the request and recorder
			handler := http.Handler(handlerToTest)
			w := tt.args.responseWriterToUse(rr)
			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expextedStatusCode, rr.Code)

		})
	}

}

func Test_egressResponseMiddleware_Request(t *testing.T) {

	type args struct {
		handler http.Handler
	}
	tests := []struct {
		name               string
		args               args
		expextedStatusCode int
		expectedHeaders    http.Header
		expectedBody       string
	}{
		{
			name: "basic success",
			args: args{
				handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					crw, ok := w.(*GechologResponseWriter)
					if !ok {
						http.Error(w, "Internal Server Error", http.StatusInternalServerError)
						return
					}
					crw.egressHeaders = http.Header{
						"Content-Type": []string{"application/json"},
					}
					crw.egressBody.Write([]byte("test"))
					crw.egressStatusCode = http.StatusOK
				}),
			},
			expextedStatusCode: http.StatusOK,
			expectedHeaders:    http.Header{"Content-Type": []string{"application/json"}},
			expectedBody:       "test",
		},
		{
			name: "failed write",
			args: args{
				handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					crw, ok := w.(*GechologResponseWriter)
					if !ok {
						http.Error(w, "Internal Server Error", http.StatusInternalServerError)
						return
					}
					crw.egressHeaders = http.Header{
						"Content-Type": []string{"application/json"},
					}
					crw.egressBody = nil // This is the error
					crw.egressStatusCode = http.StatusOK
				}),
			},
			expextedStatusCode: http.StatusInternalServerError,
			expectedHeaders:    http.Header{},
			expectedBody:       "Internal Server Error\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Create a ResponseRecorder to record the response
			rr := httptest.NewRecorder()
			g := &GechologResponseWriter{
				ResponseWriter: rr,
				egressHeaders:  http.Header{},
				egressBody:     bytes.NewBufferString(""),
				inboundBody:    bytes.NewBufferString(""),
			}

			req, err := http.NewRequest("POST", "/", nil)
			if err != nil {
				t.Fatal(err)
			}

			// Create a handler and pass the request and recorder
			handler := http.Handler(egressResponseMiddleware(tt.args.handler))
			handler.ServeHTTP(g, req)

			assert.Equal(t, tt.expextedStatusCode, rr.Code)
			for k, v := range tt.expectedHeaders {
				assert.Equal(t, v, g.egressHeaders[k])
			}
			assert.Equal(t, tt.expectedBody, rr.Body.String())

		})
	}

}

func Test_ingressEgressHeaderMiddlewareFunc_BadArgs(t *testing.T) {
	type args struct {
		requiredIngressHeaders protectedheader.ProtectedHeader
		removeHeadersMap       map[string]struct{}
		maskedHeadersMap       map[string]struct{}
		sessionHeader          string
		s                      *state
	}
	tests := []struct {
		name        string
		args        args
		expextedNil bool
	}{
		{
			name: "basic success",
			args: args{
				requiredIngressHeaders: protectedheader.ProtectedHeader{},
				removeHeadersMap:       map[string]struct{}{},
				maskedHeadersMap:       map[string]struct{}{},
				sessionHeader:          "Session-Header",
				s: &state{
					m: &sync.Mutex{},
				},
			},
			expextedNil: false,
		},
		{
			name: "requiredIngressHeaders is nil",
			args: args{
				requiredIngressHeaders: nil, // This is the error
				removeHeadersMap:       map[string]struct{}{},
				maskedHeadersMap:       map[string]struct{}{},
				sessionHeader:          "Session-Header",
				s: &state{
					m: &sync.Mutex{},
				},
			},
			expextedNil: true,
		},
		{
			name: "removeHeadersMap is nil",
			args: args{
				requiredIngressHeaders: protectedheader.ProtectedHeader{},
				removeHeadersMap:       nil,
				maskedHeadersMap:       map[string]struct{}{},
				sessionHeader:          "Session-Header",
				s: &state{
					m: &sync.Mutex{},
				},
			},
			expextedNil: true,
		},
		{
			name: "maskedHeadersMap is nil",
			args: args{
				requiredIngressHeaders: protectedheader.ProtectedHeader{},
				removeHeadersMap:       map[string]struct{}{},
				maskedHeadersMap:       nil,
				sessionHeader:          "Session-Header",
				s: &state{
					m: &sync.Mutex{},
				},
			},
			expextedNil: true,
		},
		{
			name: "sessionHeader is empty",
			args: args{
				requiredIngressHeaders: protectedheader.ProtectedHeader{},
				removeHeadersMap:       map[string]struct{}{},
				maskedHeadersMap:       map[string]struct{}{},
				sessionHeader:          "",
				s: &state{
					m: &sync.Mutex{},
				},
			},
			expextedNil: true,
		},
		{
			name: "s is nil",
			args: args{
				requiredIngressHeaders: protectedheader.ProtectedHeader{},
				removeHeadersMap:       map[string]struct{}{},
				maskedHeadersMap:       map[string]struct{}{},
				sessionHeader:          "Session-Header",
				s:                      nil,
			},
			expextedNil: true,
		},
		{
			name: "s mutex is nil",
			args: args{
				requiredIngressHeaders: protectedheader.ProtectedHeader{},
				removeHeadersMap:       map[string]struct{}{},
				maskedHeadersMap:       map[string]struct{}{},
				sessionHeader:          "Session-Header",
				s: &state{
					m: nil, // This is the error
				},
			},
			expextedNil: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := ingressEgressHeaderMiddlewareFunc(tt.args.requiredIngressHeaders, tt.args.removeHeadersMap, tt.args.maskedHeadersMap, tt.args.sessionHeader, tt.args.s)
			if tt.expextedNil {
				assert.Nil(t, f)
				return
			}
			assert.NotNil(t, f)

		})
	}
}

func Test_ingressEgressHeaderMiddlewareFunc_GechologResponseWriter(t *testing.T) {

	middleware := ingressEgressHeaderMiddlewareFunc(
		protectedheader.ProtectedHeader{},
		map[string]struct{}{},
		map[string]struct{}{},
		"Session-Header",
		&state{
			m: &sync.Mutex{},
		})
	assert.NotNil(t, middleware)

	handlerToTest := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	assert.NotNil(t, handlerToTest)

	type args struct {
		responseWriterToUse func(*httptest.ResponseRecorder) http.ResponseWriter
	}
	tests := []struct {
		name               string
		args               args
		expextedStatusCode int
	}{
		{
			name: "basic success",
			args: args{
				responseWriterToUse: func(rr *httptest.ResponseRecorder) http.ResponseWriter {
					g := &GechologResponseWriter{
						ResponseWriter:      rr,
						inboundBody:         bytes.NewBufferString(""),
						inboundHeaders:      http.Header{},
						egressHeaders:       http.Header{},
						requestObject:       gechologobject.New(),
						requestErrorObject:  gechologobject.New(),
						responseObject:      gechologobject.New(),
						responseErrorObject: gechologobject.New(),
					}
					return g
				},
			},
			expextedStatusCode: http.StatusOK,
		},
		{
			name: "not a GechologResponseWriter",
			args: args{
				responseWriterToUse: func(rr *httptest.ResponseRecorder) http.ResponseWriter {
					return http.ResponseWriter(rr)
				},
			},
			expextedStatusCode: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			req, err := http.NewRequest("POST", "/", nil)
			if err != nil {
				t.Fatal(err)
			}

			// Create a ResponseRecorder to record the response
			rr := httptest.NewRecorder()

			// Create a handler and pass the request and recorder
			handler := http.Handler(handlerToTest)
			w := tt.args.responseWriterToUse(rr)
			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expextedStatusCode, rr.Code)

		})
	}

}

func Test_ingressEgressHeaderMiddlewareFunc_RequiredHeaders(t *testing.T) {

	type args struct {
		handler                http.Handler
		requestHeaders         http.Header
		requiredIngressHeaders protectedheader.ProtectedHeader
	}
	tests := []struct {
		name               string
		args               args
		expextedStatusCode int
	}{
		{
			name: "basic success",
			args: args{
				handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}),
				requestHeaders: http.Header{
					"Content-Type": []string{"application/json"},
				},
				requiredIngressHeaders: protectedheader.ProtectedHeader{
					"Content-Type": []string{"application/json"},
				},
			},
			expextedStatusCode: http.StatusOK,
		},
		{
			name: "one any-nonzero success",
			args: args{
				handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}),
				requestHeaders: http.Header{
					"Content-Type": []string{"application/json"},
				},
				requiredIngressHeaders: protectedheader.ProtectedHeader{
					"Content-Type": []string{"regex:.+"},
				},
			},
			expextedStatusCode: http.StatusOK,
		},
		{
			name: "two headers success",
			args: args{
				handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}),
				requestHeaders: http.Header{
					"Content-Type":  []string{"application/json"},
					"Authorization": []string{"Bearer token"},
				},
				requiredIngressHeaders: protectedheader.ProtectedHeader{
					"Content-Type":  []string{"application/json"},
					"Authorization": []string{"Bearer token"},
				},
			},
			expextedStatusCode: http.StatusOK,
		},
		{
			name: "two headers, one bad value",
			args: args{
				handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}),
				requestHeaders: http.Header{
					"Content-Type":  []string{"application/xml"},
					"Authorization": []string{"Bearer token"},
				},
				requiredIngressHeaders: protectedheader.ProtectedHeader{
					"Content-Type":  []string{"application/json"},
					"Authorization": []string{"Bearer token"},
				},
			},
			expextedStatusCode: http.StatusUnauthorized,
		},
		{
			name: "header missing",
			args: args{
				handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}),
				requestHeaders: http.Header{
					// This is the error
				},
				requiredIngressHeaders: protectedheader.ProtectedHeader{
					"Content-Type": []string{"application/json"},
				},
			},
			expextedStatusCode: http.StatusUnauthorized,
		},
		{
			name: "bad header value",
			args: args{
				handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}),
				requestHeaders: http.Header{
					"Content-Type": []string{"application/xml"}, // This is the error
				},
				requiredIngressHeaders: protectedheader.ProtectedHeader{
					"Content-Type": []string{"application/json"},
				},
			},
			expextedStatusCode: http.StatusUnauthorized,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			middleware := ingressEgressHeaderMiddlewareFunc(
				tt.args.requiredIngressHeaders,
				map[string]struct{}{},
				map[string]struct{}{},
				"Session-Header",
				&state{
					m: &sync.Mutex{},
				})
			if middleware == nil {
				t.Fatal("middleware is nil")
			}

			handlerToTest := egressResponseMiddleware(middleware(tt.args.handler))
			if handlerToTest == nil {
				t.Fatal("handlerToTest is nil")
			}

			// Create a ResponseRecorder to record the response
			rr := httptest.NewRecorder()
			g := &GechologResponseWriter{
				ResponseWriter:      rr,
				inboundBody:         bytes.NewBufferString(""),
				inboundHeaders:      http.Header{},
				egressHeaders:       http.Header{},
				egressBody:          bytes.NewBufferString(""),
				requestObject:       gechologobject.New(),
				requestErrorObject:  gechologobject.New(),
				responseObject:      gechologobject.New(),
				responseErrorObject: gechologobject.New(),
			}

			req, err := http.NewRequest("POST", "/", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header = tt.args.requestHeaders

			// Create a handler and pass the request and recorder
			handler := http.Handler(handlerToTest)
			handler.ServeHTTP(g, req)

			assert.Equal(t, tt.expextedStatusCode, rr.Code)

		})
	}

}
