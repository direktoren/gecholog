package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/direktoren/gecholog/internal/gechologobject"
	"github.com/direktoren/gecholog/internal/processorconfiguration"
	"github.com/direktoren/gecholog/internal/protectedheader"
	"github.com/direktoren/gecholog/internal/router"
	"github.com/direktoren/gecholog/internal/sessionid"
	"github.com/direktoren/gecholog/internal/store"
	"github.com/direktoren/gecholog/internal/timer"
	"github.com/nats-io/nats.go"
)

type GechologResponseWriter struct {
	http.ResponseWriter

	outboundBody    *bytes.Buffer
	outboundHeaders http.Header
	outboundURL     url.URL

	inboundBody       *bytes.Buffer
	inboundHeaders    http.Header
	inboundStatusCode int

	egressBody       *bytes.Buffer
	egressHeaders    http.Header
	egressStatusCode int

	ingressOutboundTimer timer.Timer
	outboundInboundTimer timer.Timer
	inboundEgressTimer   timer.Timer
	egressPostTimer      timer.Timer

	ingressEgressTimer timer.Timer

	processorLogsRequestSync   map[string]processorLog
	processorLogsRequestAsync  map[string]processorLog
	processorLogsResponseSync  map[string]processorLog
	processorLogsResponseAsync map[string]processorLog

	requestObject      gechologobject.GechoLogObject
	requestErrorObject gechologobject.GechoLogObject

	responseObject      gechologobject.GechoLogObject
	responseErrorObject gechologobject.GechoLogObject

	rootObject      gechologobject.GechoLogObject
	rootErrorObject gechologobject.GechoLogObject

	transactionID string
	sessionID     string
}

type state struct {
	countRequests uint64
	m             *sync.Mutex
}

func (s *state) tick() uint64 {
	s.m.Lock()
	s.countRequests++
	s.m.Unlock()
	return s.countRequests
}

/*
func (s *state) get() uint64 {
	s.m.Lock()
	defer s.m.Unlock()
	return s.countRequests
}

func (s *state) now() time.Time {
	s.m.Lock()
	defer s.m.Unlock()
	return time.Now()
}*/

/*
func (w *GechologResponseWriter) Write(b []byte) (int, error) {
	w.inboundBody.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *GechologResponseWriter) WriteHeader(statusCode int) {
	w.inboundStatusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}*/

type loggingPostProcessor func(crw *GechologResponseWriter)

func loggingPostProcessorFunc(nc *nats.Conn, subject string, recordLastError func(error, time.Time), recordLastTransaction func(string), postRequestResponseProcessors http.Handler, logFilter finalLogger, s *state) loggingPostProcessor {
	if nc == nil {
		logger.Error("nats.Conn is nil")
		return nil
	}

	if subject == "" {
		logger.Error("subject is empty")
		return nil
	}

	if recordLastError == nil {
		logger.Error("func astError is nil")
		return nil
	}

	if recordLastTransaction == nil {
		logger.Error("func lastTransactionID is nil")
		return nil
	}

	if s == nil {
		logger.Error("state is nil")
		return nil
	}

	if s.m == nil {
		logger.Error("state mutex is nil")
		return nil
	}

	if postRequestResponseProcessors == nil {
		logger.Error("postHandler is nil")
		return nil
	}
	return func(crw *GechologResponseWriter) {

		dummy, _ := http.NewRequest("GET", "/", nil)

		// Post Processing Processors are called here
		postRequestResponseProcessors.ServeHTTP(crw, dummy)

		// -------------- Finalize requestObject --------------

		finalOutboundHeaders := protectedheader.ProtectedHeader(crw.outboundHeaders)
		store.Store(&crw.requestObject, &crw.requestErrorObject, "outbound_headers", &finalOutboundHeaders)

		store.Store(&crw.requestObject, &crw.requestErrorObject, "url", crw.outboundURL.String())

		store.Store(&crw.requestObject, &crw.requestErrorObject, "ingress_outbound_timer", crw.ingressOutboundTimer)

		store.StoreInArray(&crw.requestObject, &crw.requestErrorObject, "processors", func(m map[string]processorLog) *gechologobject.GechoLogObject {
			tmpObject := gechologobject.New()
			for processor, stats := range m {
				tmpObject.AssignField(processor, &stats)
			}

			return &tmpObject
		}(crw.processorLogsRequestSync))

		store.StoreInArray(&crw.requestObject, &crw.requestErrorObject, "processors_async", func(m map[string]processorLog) *gechologobject.GechoLogObject {
			tmpObject := gechologobject.New()
			for processor, stats := range m {
				tmpObject.AssignField(processor, &stats)
			}

			return &tmpObject
		}(crw.processorLogsRequestAsync))

		store.StoreInArray(&crw.requestObject, &crw.requestErrorObject, "error", &crw.requestErrorObject)

		// -------------- Finalize responseObject --------------

		crw.inboundEgressTimer.SetStart(crw.outboundInboundTimer.GetStop())
		crw.inboundEgressTimer.SetStop(crw.ingressEgressTimer.GetStop())

		store.Store(&crw.responseObject, &crw.responseErrorObject, "inbound_egress_timer", crw.inboundEgressTimer)
		store.Store(&crw.responseObject, &crw.responseErrorObject, "outbound_inbound_timer", crw.outboundInboundTimer)

		store.StoreInArray(&crw.responseObject, &crw.responseErrorObject, "processors", func(m map[string]processorLog) *gechologobject.GechoLogObject {
			tmpObject := gechologobject.New()
			for processor, stats := range m {
				tmpObject.AssignField(processor, &stats)
			}

			return &tmpObject
		}(crw.processorLogsResponseSync))

		store.StoreInArray(&crw.responseObject, &crw.responseErrorObject, "processors_async", func(m map[string]processorLog) *gechologobject.GechoLogObject {
			tmpObject := gechologobject.New()
			for processor, stats := range m {
				tmpObject.AssignField(processor, &stats)
			}

			return &tmpObject
		}(crw.processorLogsResponseAsync))

		store.StoreInArray(&crw.responseObject, &crw.responseErrorObject, "error", &crw.responseErrorObject)

		// -------------- Populate root Object --------------

		store.Store(&crw.rootObject, &crw.rootErrorObject, "session_id", json.RawMessage("\""+crw.sessionID+"\""))
		store.Store(&crw.rootObject, &crw.rootErrorObject, "transaction_id", json.RawMessage("\""+crw.transactionID+"\""))

		// Apply the final logger filters to requestObject

		requestObjectFiltered := gechologobject.New()
		requestObjectFiltered = gechologobject.AppendNew(requestObjectFiltered, crw.requestObject)
		if len(logFilter.Request.FieldsInclude) != 0 {
			requestObjectFiltered = gechologobject.Filter(requestObjectFiltered, logFilter.Request.FieldsInclude)
		}
		if len(logFilter.Request.FieldsExclude) != 0 {
			blockedMap := make(map[string]struct{})
			for _, k := range logFilter.Request.FieldsExclude {
				blockedMap[k] = struct{}{}
			}
			allFieldNames := requestObjectFiltered.FieldNames()
			filteredFieldNames := []string{}
			for _, k := range allFieldNames {
				if _, blocked := blockedMap[k]; !blocked {
					filteredFieldNames = append(filteredFieldNames, k)
				}
			}
			requestObjectFiltered = gechologobject.Filter(requestObjectFiltered, filteredFieldNames)
		}
		store.Store(&crw.rootObject, &crw.rootErrorObject, "request", &requestObjectFiltered)

		// Apply the final logger filters to responseObject

		responseObjectFiltered := gechologobject.New()
		responseObjectFiltered = gechologobject.AppendNew(responseObjectFiltered, crw.responseObject)
		if len(logFilter.Response.FieldsInclude) != 0 {
			responseObjectFiltered = gechologobject.Filter(responseObjectFiltered, logFilter.Response.FieldsInclude)
		}
		if len(logFilter.Response.FieldsExclude) != 0 {
			blockedMap := make(map[string]struct{})
			for _, k := range logFilter.Response.FieldsExclude {
				blockedMap[k] = struct{}{}
			}
			allFieldNames := responseObjectFiltered.FieldNames()
			filteredFieldNames := []string{}
			for _, k := range allFieldNames {
				if _, blocked := blockedMap[k]; !blocked {
					filteredFieldNames = append(filteredFieldNames, k)
				}
			}
			responseObjectFiltered = gechologobject.Filter(responseObjectFiltered, filteredFieldNames)
		}
		store.Store(&crw.rootObject, &crw.rootErrorObject, "response", &responseObjectFiltered)
		store.Store(&crw.rootObject, &crw.rootErrorObject, "ingress_egress_timer", crw.ingressEgressTimer)

		crw.egressPostTimer.SetStart(crw.ingressEgressTimer.GetStop())
		crw.egressPostTimer.Stop() // <------- POST POINT

		store.Store(&crw.rootObject, &crw.rootErrorObject, "egress_post_timer", crw.egressPostTimer)
		store.StoreInArray(&crw.rootObject, &crw.rootErrorObject, "error", &crw.rootErrorObject)

		rootBytes, err := json.Marshal(&crw.rootObject)
		if err != nil {
			logger.Error(
				"unexpected json.Marshal error",
				slog.String("context", "post"),
				slog.String("transaction_id", crw.transactionID),
				slog.Any("error", err),
			)

			recordLastError(fmt.Errorf("logger: Unexpected json.Marshal error %v", err), time.Now())
			return
		}

		// Push it to the bus
		err = nc.Publish(subject, rootBytes)
		if err != nil {
			logger.Error(
				"problem sending log to service bus",
				slog.String("context", "post"),
				slog.String("transaction_id", crw.transactionID),
				slog.Any("error", err),
			)

			recordLastError(fmt.Errorf("logger: Problem sending log to service bus: %v", err), time.Now())

			return
		}

		recordLastTransaction(crw.transactionID) // Only written if all states completed
		logger.Debug(
			"logger completed",
			slog.String("context", "post"),
			slog.String("transaction_id", crw.transactionID),
		)

	}
}
func loggingMiddlewareFunc(postProcessingAndFinalizeLogging loggingPostProcessor, logUnauthorized bool) func(http.Handler) http.Handler {
	if postProcessingAndFinalizeLogging == nil {
		logger.Error("postProcessingAndFinalizeLogging is nil")
		return nil
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			crw := &GechologResponseWriter{
				ResponseWriter: w,

				outboundBody:    bytes.NewBufferString(""),
				outboundHeaders: http.Header{},
				outboundURL:     url.URL{},

				inboundBody:       bytes.NewBufferString(""),
				inboundHeaders:    http.Header{},
				inboundStatusCode: http.StatusOK,

				egressBody:       bytes.NewBufferString(""),
				egressHeaders:    http.Header{},
				egressStatusCode: http.StatusOK,

				ingressOutboundTimer: timer.Timer{},
				outboundInboundTimer: timer.Timer{},
				inboundEgressTimer:   timer.Timer{},
				egressPostTimer:      timer.Timer{},

				ingressEgressTimer: timer.Timer{},

				processorLogsRequestSync:   map[string]processorLog{},
				processorLogsRequestAsync:  map[string]processorLog{},
				processorLogsResponseSync:  map[string]processorLog{},
				processorLogsResponseAsync: map[string]processorLog{},

				requestObject:      gechologobject.New(),
				requestErrorObject: gechologobject.New(),

				responseObject:      gechologobject.New(),
				responseErrorObject: gechologobject.New(),

				rootObject:      gechologobject.New(),
				rootErrorObject: gechologobject.New(),
			}

			crw.ingressEgressTimer.Start()

			// Call the next handler in the chain
			next.ServeHTTP(crw, r)

			crw.ingressEgressTimer.Stop()

			if crw.egressStatusCode == http.StatusUnauthorized && !logUnauthorized {
				return
			}
			go postProcessingAndFinalizeLogging(crw)

		})
	}
}

func sessionMiddlewareFunc(gatewayID string, sesssionHeader string, s *state) func(http.Handler) http.Handler {

	if gatewayID == "" {
		logger.Error("gatewayID is empty")
		return nil
	}

	ok := sessionid.ValidateGatewayID(gatewayID)
	if !ok {
		logger.Error("Invalid GatewayID", slog.String("gatewayID", gatewayID))
		return nil
	}

	if sesssionHeader == "" {
		logger.Error("sessionHeader is empty")
		return nil
	}

	if s == nil {
		logger.Error("state is nil")
		return nil
	}

	if s.m == nil {
		logger.Error("state mutex is nil")
		return nil
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			crw, ok := w.(*GechologResponseWriter)
			if !ok {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				logger.Error("failed to cast ResponseWriter to GechologResponseWriter")
				return
			}

			sessionID, transactionID, err := sessionid.Update(r.Header.Get(sesssionHeader))
			if err != nil {
				sessionID, err = sessionid.Generate(gatewayID, crw.ingressEgressTimer.GetStart(), s.tick())
				if err != nil {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					logger.Error("failed to generate sessionID", slog.Any("error", err))
					return
				}
				transactionID = sessionID
			}

			crw.transactionID = transactionID
			crw.sessionID = sessionID

			next.ServeHTTP(crw, r)

		})
	}
}

func egressResponseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		crw, ok := w.(*GechologResponseWriter)
		if !ok {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			logger.Error("failed to cast ResponseWriter to GechologResponseWriter")
			return
		}

		next.ServeHTTP(crw, r)

		for key, values := range crw.egressHeaders {
			for _, value := range values {
				crw.Header().Add(key, value)
			}
		}

		if crw.egressBody == nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			logger.Error("egress body is nil")
			return
		}

		crw.Header().Set("Content-Length", fmt.Sprintf("%d", crw.egressBody.Len()))
		crw.WriteHeader(crw.egressStatusCode)
		_, err := io.Copy(crw, crw.egressBody)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			logger.Error("failed to write body", slog.Any("error", err))
			return
		}

	})
}

func ingressEgressHeaderMiddlewareFunc(requiredIngressHeaders protectedheader.ProtectedHeader, removeHeadersMap map[string]struct{}, maskedHeadersMap map[string]struct{}, sessionHeader string, s *state) func(http.Handler) http.Handler {

	if requiredIngressHeaders == nil {
		logger.Error("requiredHeaders is empty")
		return nil
	}

	if removeHeadersMap == nil {
		logger.Error("removeHeadersMap is empty")
		return nil
	}

	if maskedHeadersMap == nil {
		logger.Error("maskedHeadersMap is empty")
		return nil
	}

	if sessionHeader == "" {
		logger.Error("sessionHeader is empty")
		return nil
	}

	if s == nil {
		logger.Error("state is nil")
		return nil
	}

	if s.m == nil {
		logger.Error("state mutex is nil")
		return nil
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			crw, ok := w.(*GechologResponseWriter)
			if !ok {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				logger.Error("failed to cast ResponseWriter to GechologResponseWriter")
				return
			}

			// Use Protect Headers since they can contain sensitive information
			ingressHeaders := protectedheader.ProtectedHeader{}
			ingressHeaders = protectedheader.AppendNew(ingressHeaders, protectedheader.ProtectedHeader(r.Header))

			// ------------ ingress header check (like API key) ------------
			negativeSet := protectedheader.Remove(ingressHeaders, requiredIngressHeaders.GetHeaderList())
			candidateSet := protectedheader.Remove(ingressHeaders, negativeSet.GetHeaderList())

			store.Store(&crw.requestObject, &crw.requestErrorObject, "ingress_headers", &ingressHeaders)

			egressHeaders := protectedheader.ProtectedHeader{}
			egressHeaders[sessionHeader] = []string{crw.transactionID} // Add the session header.
			defer func() {
				// Now write the rest of the headers
				for key, values := range egressHeaders {
					for _, value := range values {
						crw.egressHeaders.Add(key, value)
					}
				}
				finalEgressHeaders := protectedheader.ProtectedHeader(crw.egressHeaders) // what you see is what you get (except if its masked)
				store.Store(&crw.responseObject, &crw.responseErrorObject, "egress_headers", &finalEgressHeaders)
			}()

			if ok, unauthorizedMsg := protectedheader.EqualIfNonEmptyExistsIfCatchall(requiredIngressHeaders, candidateSet); !ok {
				crw.egressBody.Write([]byte(`{"error":"unauthorized ` + unauthorizedMsg + `"}`))
				crw.egressStatusCode = http.StatusUnauthorized

				crw.requestErrorObject.AssignField("ingress_headers", "Unauthorized:"+unauthorizedMsg)
				return
			}

			store.Store(&crw.requestObject, &crw.requestErrorObject, "outbound_headers", &ingressHeaders)

			next.ServeHTTP(crw, r)

			// Extract egress headers from responseObject, for example from processing
			egressHeaders = func() protectedheader.ProtectedHeader {
				egressHeadersRaw, err := crw.responseObject.GetField("egress_headers")
				if err != nil {
					// Not there
					return protectedheader.ProtectedHeader{}
				}

				parsedHeaders := protectedheader.ProtectedHeader{}
				err = json.Unmarshal(egressHeadersRaw, &parsedHeaders) // populate headers
				if err != nil {
					crw.responseErrorObject.AssignField("egress_headers", err.Error())
					return protectedheader.ProtectedHeader{}
				}
				return parsedHeaders
			}()

			// Remove headers from egressHeaders according to removeHeadersMap
			removedHeaders := protectedheader.ProtectedHeader{}
			for key, _ := range removeHeadersMap {
				_, exists := egressHeaders[key]
				if exists {
					delete(egressHeaders, key)
					removedHeaders[key] = []string{}
				}
			}
			if len(removedHeaders) > 0 {
				store.Store(&crw.responseObject, &crw.responseErrorObject, "egresss_headers_removed", &removedHeaders)
			}

			// Populate the discarded headers
			inboundHeaders := func() protectedheader.ProtectedHeader {
				egressHeadersRaw, err := crw.responseObject.GetField("inbound_headers")
				if err != nil {
					// Not there
					return protectedheader.ProtectedHeader{}
				}

				parsedHeaders := protectedheader.ProtectedHeader{}
				err = json.Unmarshal(egressHeadersRaw, &parsedHeaders) // populate headers
				if err != nil {
					crw.responseErrorObject.AssignField("inbound_headers", err.Error())
					return protectedheader.ProtectedHeader{}
				}
				return parsedHeaders
			}()

			discatedHeaders := protectedheader.ProtectedHeader{}
			for key, _ := range inboundHeaders {
				if _, exists := egressHeaders[key]; !exists {
					if _, exists := removedHeaders[key]; !exists {
						discatedHeaders[key] = []string{}
					}
				}
			}

			if len(discatedHeaders) > 0 {
				store.Store(&crw.responseObject, &crw.responseErrorObject, "egresss_headers_discarded", &discatedHeaders)
			}

			// We remove the masked values (ie the headers in maskedHeadersMap) to avoid overwriting them
			egressHeaders = protectedheader.Remove(egressHeaders, maskedHeadersMap)
			egressHeaders[sessionHeader] = []string{crw.transactionID} // Add (back) the session header. If it was changed by processing, it will be restored

			// Time to prepare headers for the response

			// Start by writing the masked headers in unmasked format
			for key, values := range crw.inboundHeaders {
				if _, masked := maskedHeadersMap[key]; masked {
					for _, value := range values {
						crw.egressHeaders.Add(key, value)
					}
				}
			}

		})
	}
}

func ingressEgressPayloadMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		crw, ok := w.(*GechologResponseWriter)
		if !ok {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			logger.Error("failed to cast ResponseWriter to GechologResponseWriter")
			return
		}

		ingressBytes, err := io.ReadAll(r.Body)
		if err != nil {
			crw.egressBody.Write([]byte(`{"error":"internal server error"}`))
			crw.egressStatusCode = http.StatusInternalServerError

			logger.Error("failed to read body", slog.Any("error", err))
			return
		}
		if !json.Valid(ingressBytes) {
			ingressBytes = []byte(fmt.Sprintf("%q", ingressBytes))
			crw.requestErrorObject.AssignField("ingress_payload", "Payload not a valid json")
		}

		// Populate ingress_payload field so we can send to processors
		store.Store(&crw.requestObject, &crw.requestErrorObject, "ingress_payload", ingressBytes)
		store.Store(&crw.requestObject, &crw.requestErrorObject, "outbound_payload", ingressBytes)

		next.ServeHTTP(crw, r)

		if crw.egressBody.Len() == 0 {
			// If egress body is not already set, we fetch it from the responseObject
			egressBytes, err := crw.responseObject.GetField("egress_payload")
			if err != nil {
				crw.responseErrorObject.AssignField("egress_payload", err.Error())
				egressBytes = json.RawMessage(`{"error":"internal server error"}`)
				crw.egressStatusCode = http.StatusInternalServerError
			}
			_, err = crw.egressBody.Write(egressBytes)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				crw.responseErrorObject.AssignField("egress_payload", err.Error())

				logger.Error("failed to write body", slog.Any("error", err))
				return
			}
		}

		defer func() {
			store.Store(&crw.responseObject, &crw.responseErrorObject, "egress_status_code", crw.egressStatusCode)
			if crw.egressStatusCode != http.StatusOK {
				crw.responseErrorObject.AssignField("egress_status_code", fmt.Sprintf("%d", crw.egressStatusCode))
			}
		}()

		if crw.egressStatusCode != http.StatusOK {
			// if egressStatusCode is already set, we don't need to do anything more
			return
		}

		egressStatusCodeRaw, err := crw.responseObject.GetField("egress_status_code")
		if err != nil {
			crw.responseErrorObject.AssignField("egress_status_code", err.Error())
			crw.egressStatusCode = http.StatusInternalServerError
			return
		}
		var egressStatusCodeTmp int
		err = json.Unmarshal(egressStatusCodeRaw, &egressStatusCodeTmp)
		if err != nil {
			crw.responseErrorObject.AssignField("egress_status_code", err.Error())
			crw.egressStatusCode = http.StatusInternalServerError
			return
		}

		crw.egressStatusCode = egressStatusCodeTmp

	})
}

func outboundInboundPayloadMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		crw, ok := w.(*GechologResponseWriter)
		if !ok {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			logger.Error("failed to cast ResponseWriter to GechologResponseWriter")
			return
		}

		payload, err := crw.requestObject.GetField("outbound_payload")
		if err != nil {
			// We don't seem to have a valid payload
			crw.egressBody.Write([]byte(`{"error":"internal server error"}`))
			crw.egressStatusCode = http.StatusInternalServerError

			crw.responseErrorObject.AssignField("outbound_payload", err.Error())
			return
		}

		if !json.Valid(payload) {
			crw.requestErrorObject.AssignField("outbound_payload", "Payload not a valid json2")
		}
		crw.outboundBody.Write(payload)

		crw.ingressOutboundTimer.SetStart(crw.ingressEgressTimer.GetStart())
		crw.ingressOutboundTimer.Stop()

		next.ServeHTTP(crw, r)

		crw.outboundInboundTimer.SetStart(crw.ingressOutboundTimer.GetStop())
		crw.outboundInboundTimer.Stop()

		store.Store(&crw.responseObject, &crw.responseErrorObject, "inbound_status_code", crw.inboundStatusCode)
		store.Store(&crw.responseObject, &crw.responseErrorObject, "egress_status_code", crw.inboundStatusCode)

		if !json.Valid(crw.inboundBody.Bytes()) {
			crw.responseErrorObject.AssignField("inbound_payload", "Payload not a valid json3")
			store.Store(&crw.responseObject, &crw.responseErrorObject, "inbound_payload", []byte(fmt.Sprintf("%q", crw.inboundBody.Bytes())))
			store.Store(&crw.responseObject, &crw.responseErrorObject, "egress_payload", []byte(fmt.Sprintf("%q", crw.inboundBody.Bytes())))
			return
		}

		store.Store(&crw.responseObject, &crw.responseErrorObject, "inbound_payload", crw.inboundBody.Bytes())
		store.Store(&crw.responseObject, &crw.responseErrorObject, "egress_payload", crw.inboundBody.Bytes())
	})
}

func controlFieldMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		crw, ok := w.(*GechologResponseWriter)
		if !ok {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)

			logger.Error("failed to cast ResponseWriter to GechologResponseWriter")
			return
		}

		controlFieldRaw, controlFieldExistsErr := crw.requestObject.GetField("control")
		if controlFieldExistsErr == nil {
			// We found the "control" field
			// Override - we don't make the outbound call

			crw.inboundBody.Write(controlFieldRaw)
			crw.inboundStatusCode = http.StatusOK

			return
		}

		next.ServeHTTP(crw, r)
	})
}

func ingressQueryParametersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		crw, ok := w.(*GechologResponseWriter)
		if !ok {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			logger.Error("failed to cast ResponseWriter to GechologResponseWriter")
			return
		}

		ingressQueryParameters := r.URL.Query()
		store.StoreInArray(&crw.requestObject, &crw.requestErrorObject, "ingress_query_parameters", func(u url.Values) *gechologobject.GechoLogObject {
			tmpObject := gechologobject.New()
			for key, value := range u {
				tmpObject.AssignField(key, &value)
			}
			return &tmpObject
		}(ingressQueryParameters))

		store.StoreInArray(&crw.requestObject, &crw.requestErrorObject, "outbound_query_parameters", func(u url.Values) *gechologobject.GechoLogObject {
			tmpObject := gechologobject.New()
			for key, value := range u {
				tmpObject.AssignField(key, &value)
			}
			return &tmpObject
		}(ingressQueryParameters))

		next.ServeHTTP(crw, r)
	})
}

func outboundQueryParametersMiddlewareFunc(outboundRouter router.OutboundNode, s *state) func(http.Handler) http.Handler {

	if s == nil {
		logger.Error("state is nil")
		return nil
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			crw, ok := w.(*GechologResponseWriter)
			if !ok {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				logger.Error("failed to cast ResponseWriter to GechologResponseWriter")
				return
			}

			// Extract query parameters from the outbound URL and endpoint
			routerParsedURL, err := url.Parse(outboundRouter.Url)
			if err != nil {
				crw.egressBody.Write([]byte(`{"error":"internal server error"}`))
				crw.egressStatusCode = http.StatusInternalServerError

				crw.requestErrorObject.AssignField("outbound_url", err.Error())
				logger.Error("failed to parse outbound url", slog.Any("error", err))
				return
			}
			routerQueryParameters := routerParsedURL.Query()

			endpointParsedURL, err := url.Parse(outboundRouter.Endpoint)
			if err != nil {
				crw.egressBody.Write([]byte(`{"error":"internal server error"}`))
				crw.egressStatusCode = http.StatusInternalServerError

				crw.requestErrorObject.AssignField("outbound_url", err.Error())
				logger.Error("failed to parse outbound url", slog.Any("error", err))
				return
			}
			endpointQueryParameters := endpointParsedURL.Query()

			outboundQueryParameters := func() url.Values {

				outboundQueryParametersRaw, err := crw.requestObject.GetField("outbound_query_parameters")
				if err != nil {
					return url.Values{}
				}
				outboundQueryParametersArray := []store.ArrayLog{}
				err = json.Unmarshal(outboundQueryParametersRaw, &outboundQueryParametersArray) // populate query parameters
				if err != nil {
					crw.requestErrorObject.AssignField("outbound_query_parameters", err.Error())
					return url.Values{}
				}
				op := url.Values{}
				for _, valueBytes := range outboundQueryParametersArray {
					values := []string{}
					err := json.Unmarshal(valueBytes.Details, &values)
					if err != nil {
						crw.requestErrorObject.AssignField("outbound_query_parameters", err.Error())
						return url.Values{}
					}
					op[valueBytes.Name] = values
				}
				return op
			}()

			overwrittenQueryParameters := url.Values{}
			discardedQueryParameters := url.Values{}

			// Then write over if there are query parameters in the outbound path
			for key, values := range routerQueryParameters {
				if _, exists := outboundQueryParameters[key]; exists {
					for _, value := range values {
						overwrittenQueryParameters.Add(key, value)
					}
					outboundQueryParameters.Del(key)
				}
				for _, value := range values {
					outboundQueryParameters.Add(key, value)
				}
				// Write over with the ones from the config file
				//tmpOutboundqueryParameters[key] = value
			}

			// Then write over if there are query parameters in the outbound endpoint
			for key, values := range endpointQueryParameters {
				if _, exists := outboundQueryParameters[key]; exists {
					if _, exists := overwrittenQueryParameters[key]; !exists {
						overwrittenQueryParameters.Del(key)
					}
					for _, value := range values {
						overwrittenQueryParameters.Add(key, value)
					}
					outboundQueryParameters.Del(key)
				}
				for _, value := range values {
					outboundQueryParameters.Set(key, value)
				}
			}
			rQuery := r.URL.Query()
			for key, values := range rQuery {
				if _, exists := outboundQueryParameters[key]; !exists {
					for _, value := range values {
						discardedQueryParameters.Add(key, value)
					}
				}
			}

			crw.outboundURL.RawQuery = outboundQueryParameters.Encode()

			store.Store(&crw.requestObject, &crw.requestErrorObject, "outbound_query_parameters", &outboundQueryParameters)
			if len(overwrittenQueryParameters) > 0 {
				store.Store(&crw.requestObject, &crw.requestErrorObject, "outbound_query_parameters_overwritten", &overwrittenQueryParameters)
			}
			if len(discardedQueryParameters) > 0 {
				store.Store(&crw.requestObject, &crw.requestErrorObject, "outbound_query_parameters_discarded", &discardedQueryParameters)
			}

			next.ServeHTTP(crw, r)
		})
	}
}
func outboundInboundHeaderMiddlewareFunc(staticOutboundHeaders protectedheader.ProtectedHeader, removeHeadersMap map[string]struct{}, maskedHeadersMap map[string]struct{}, sessionHeader string, s *state) func(http.Handler) http.Handler {

	if staticOutboundHeaders == nil {
		logger.Error("staticOutboundHeaders is empty")
		return nil
	}

	if removeHeadersMap == nil {
		logger.Error("removeHeadersMap is empty")
		return nil
	}

	if maskedHeadersMap == nil {
		logger.Error("maskedHeadersMap is empty")
		return nil
	}

	if sessionHeader == "" {
		logger.Error("sessionHeader is empty")
		return nil
	}

	if s == nil {
		logger.Error("state is nil")
		return nil
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			crw, ok := w.(*GechologResponseWriter)
			if !ok {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				logger.Error("failed to cast ResponseWriter to GechologResponseWriter")
				return
			}

			// Extract outbound headers from requestObject, after processing
			outboundProcessedHeaders := func() protectedheader.ProtectedHeader {
				outboundHeadersRaw, err := crw.requestObject.GetField("outbound_headers")
				if err != nil {
					// Not there
					return protectedheader.ProtectedHeader{}
				}

				parsedHeaders := protectedheader.ProtectedHeader{}
				err = json.Unmarshal(outboundHeadersRaw, &parsedHeaders) // populate headers
				if err != nil {
					crw.requestErrorObject.AssignField("outbound_headers", err.Error())
					return protectedheader.ProtectedHeader{}
				}
				return parsedHeaders
			}()

			outboundHeaders := protectedheader.ProtectedHeader{}
			outboundHeaders = protectedheader.AppendNew(outboundHeaders, staticOutboundHeaders)
			outboundHeaders = protectedheader.AppendNew(outboundHeaders, outboundProcessedHeaders)

			removedHeaders := protectedheader.ProtectedHeader{}
			for key, _ := range removeHeadersMap {
				_, exists := outboundHeaders[key]
				if exists {
					delete(outboundHeaders, key)
					removedHeaders[key] = []string{}
				}
			}
			if len(removedHeaders) > 0 {
				store.Store(&crw.requestObject, &crw.requestErrorObject, "outbound_headers_removed", &removedHeaders)
			}

			discatedHeaders := protectedheader.ProtectedHeader{}
			ingressHeaders := func() protectedheader.ProtectedHeader {
				outboundHeadersRaw, err := crw.requestObject.GetField("ingress_headers")
				if err != nil {
					// Not there
					return protectedheader.ProtectedHeader{}
				}

				parsedHeaders := protectedheader.ProtectedHeader{}
				err = json.Unmarshal(outboundHeadersRaw, &parsedHeaders) // populate headers
				if err != nil {
					crw.requestErrorObject.AssignField("ingress_headers", err.Error())
					return protectedheader.ProtectedHeader{}
				}
				return parsedHeaders
			}()

			for key, _ := range ingressHeaders {
				if _, exists := outboundHeaders[key]; !exists {
					if _, exists := removedHeaders[key]; !exists {
						discatedHeaders[key] = []string{}
					}
				}
			}

			if len(discatedHeaders) > 0 {
				store.Store(&crw.requestObject, &crw.requestErrorObject, "outbound_headers_discarded", &discatedHeaders)
			}

			// Add the masked ingress Headers
			for key, values := range r.Header {
				if _, masked := maskedHeadersMap[key]; masked {
					for _, value := range values {
						crw.outboundHeaders.Add(key, value)
					}
				}
			}

			// Never overwrite the masked headers
			outboundHeaders = protectedheader.Remove(outboundHeaders, maskedHeadersMap)

			// Overwrite from responseObject to http.ResponseWriter
			for key, values := range outboundHeaders {
				for _, value := range values {
					crw.outboundHeaders.Add(key, value)
				}
			}

			next.ServeHTTP(crw, r)

			inboundHeaders := protectedheader.ProtectedHeader(crw.inboundHeaders)
			store.Store(&crw.responseObject, &crw.responseErrorObject, "inbound_headers", &inboundHeaders)

			inboundHeaders[sessionHeader] = []string{crw.transactionID}
			store.Store(&crw.responseObject, &crw.responseErrorObject, "egress_headers", &inboundHeaders)

		})
	}
}

func extractData(o *gechologobject.GechoLogObject, p processorconfiguration.ProcessorConfiguration) []byte {

	objectFiltered := gechologobject.New()
	objectFiltered = gechologobject.AppendNew(objectFiltered, *o)
	if len(p.InputFieldsInclude) != 0 {
		objectFiltered = gechologobject.Filter(objectFiltered, p.InputFieldsInclude)
	}
	if len(p.InputFieldsExclude) != 0 {
		blockedMap := make(map[string]struct{})
		for _, k := range p.InputFieldsExclude {
			blockedMap[k] = struct{}{}
		}
		allFieldNames := objectFiltered.FieldNames()
		filteredFieldNames := []string{}
		for _, k := range allFieldNames {
			if _, blocked := blockedMap[k]; !blocked {
				filteredFieldNames = append(filteredFieldNames, k)
			}
		}
		objectFiltered = gechologobject.Filter(objectFiltered, filteredFieldNames)
	}

	//o.Filter(op.InputFieldsInclude, op.InputFieldsExclude)
	if objectFiltered.IsEmpty() {
		return []byte{}
	}

	processorPayloadBytes, err := json.Marshal(&objectFiltered)
	if err != nil {
		logger.Error(
			"prepare error",
			slog.String("filtered_object", objectFiltered.DebugString()),
			slog.Any("error", err),
		)
		return []byte{}
	}

	return processorPayloadBytes
}

func writeData(o *gechologobject.GechoLogObject, e *gechologobject.GechoLogObject, processor processorconfiguration.ProcessorConfiguration, data []byte) bool {

	var object gechologobject.GechoLogObject
	err := json.Unmarshal(data, &object)
	if err != nil {
		e.AssignField(processor.Name, err.Error())
		return false
	}

	if object.IsEmpty() {
		return false
	}

	objectFiltered := gechologobject.New()
	objectFiltered = gechologobject.AppendNew(objectFiltered, object)
	objectFiltered = gechologobject.Filter(objectFiltered, processor.OutputFieldsWrite)

	if processor.Modifier {
		(*o) = gechologobject.Replace(*o, objectFiltered)
		return true
	}
	// Type is "annotator"
	(*o) = gechologobject.AppendNew(*o, objectFiltered)
	return true
}

type processorsMiddlewareFunc func(ctx context.Context, nc *nats.Conn, processor []processorconfiguration.ProcessorConfiguration, s *state) func(http.Handler) http.Handler

func requestProcessorMiddlewareFunc(ctx context.Context, nc *nats.Conn, processor []processorconfiguration.ProcessorConfiguration, s *state) func(http.Handler) http.Handler {

	if nc == nil {
		logger.Error("nats.Conn is empty")
		return nil
	}

	if len(processor) == 0 {
		logger.Error("processor is empty")
		return nil
	}

	if s == nil {
		logger.Error("state is nil")
		return nil
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			crw, ok := w.(*GechologResponseWriter)
			if !ok {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				logger.Error("failed to cast ResponseWriter to GechologResponseWriter")
				return
			}

			logEntries := make([]processorLog, len(processor))
			for log, _ := range logEntries {
				logEntries[log] = processorLog{
					Required:  processor[log].Required,
					Completed: false,
				}
			}

			data := make([][]byte, len(processor))
			for i, p := range processor {
				data[i] = extractData(&crw.requestObject, p)
			}

			wg := sync.WaitGroup{}
			m := sync.Mutex{}
			for i, p := range processor {
				if len(data[i]) != 0 {
					wg.Add(1)
					go func(i int, p processorconfiguration.ProcessorConfiguration) {

						defer wg.Done()
						response := func() []byte {
							logEntries[i].Timestamp.Start()
							defer logEntries[i].Timestamp.Stop()
							logger.Debug("timeout", slog.String("processor", p.Name), slog.Any("timeout", p.Timeout), slog.Any("duration", time.Duration(p.Timeout)*time.Millisecond))
							ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(p.Timeout)*time.Millisecond)
							defer cancel()
							msg, err := nc.RequestWithContext(ctxTimeout, p.ServiceBusTopic, data[i])
							if err != nil {
								logger.Error("failed request processor", slog.Any("error", err))
								m.Lock()
								crw.requestErrorObject.AssignField(p.Name, err.Error())
								m.Unlock()
								return []byte{}
							}
							return msg.Data
						}()
						if len(response) != 0 {
							m.Lock()
							logEntries[i].Completed = writeData(&crw.requestObject, &crw.requestErrorObject, p, response)
							m.Unlock()
						}
					}(i, p)
				}
			}
			wg.Wait()

			for i, p := range processor {
				if !logEntries[i].Completed && p.Required {
					crw.requestErrorObject.AssignField(p.Name, "required processor didn't run")
					return
				}
				switch p.Async {
				case true:
					crw.processorLogsRequestAsync[p.Name] = logEntries[i]
				case false:
					crw.processorLogsRequestSync[p.Name] = logEntries[i]
				}
			}

			next.ServeHTTP(crw, r)

		})
	}
}

func responseProcessorMiddlewareFunc(ctx context.Context, nc *nats.Conn, processor []processorconfiguration.ProcessorConfiguration, s *state) func(http.Handler) http.Handler {

	if nc == nil {
		logger.Error("nats.Conn is empty")
		return nil
	}

	if len(processor) == 0 {
		logger.Error("processor is empty")
		return nil
	}

	if s == nil {
		logger.Error("state is nil")
		return nil
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			crw, ok := w.(*GechologResponseWriter)
			if !ok {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				logger.Error("failed to cast ResponseWriter to GechologResponseWriter")
				return
			}

			next.ServeHTTP(crw, r)

			logEntries := make([]processorLog, len(processor))
			for log, _ := range logEntries {
				logEntries[log] = processorLog{
					Required:  processor[log].Required,
					Completed: false,
				}
			}

			data := make([][]byte, len(processor))
			for i, p := range processor {
				data[i] = extractData(&crw.responseObject, p)
			}

			wg := sync.WaitGroup{}
			m := sync.Mutex{}
			for i, p := range processor {
				if len(data[i]) != 0 {
					wg.Add(1)
					go func(i int, p processorconfiguration.ProcessorConfiguration) {

						defer wg.Done()
						response := func() []byte {
							logEntries[i].Timestamp.Start()
							defer logEntries[i].Timestamp.Stop()
							logger.Debug("timeout", slog.String("processor", p.Name), slog.Any("timeout", p.Timeout), slog.Any("duration", time.Duration(p.Timeout)*time.Millisecond), slog.String("now", time.Now().String()))
							ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(p.Timeout)*time.Millisecond)
							defer cancel()
							msg, err := nc.RequestWithContext(ctxTimeout, p.ServiceBusTopic, data[i])
							logger.Debug("after", slog.String("processor", p.Name), slog.Any("timeout", p.Timeout), slog.Any("duration", time.Duration(p.Timeout)*time.Millisecond), slog.String("now", time.Now().String()))

							if err != nil {
								logger.Error("failed response processor", slog.Any("error", err))
								m.Lock()
								crw.responseErrorObject.AssignField(p.Name, err.Error())
								m.Unlock()
								return []byte{}
							}
							return msg.Data
						}()
						if len(response) != 0 {
							m.Lock()
							logEntries[i].Completed = writeData(&crw.responseObject, &crw.responseErrorObject, p, response)
							m.Unlock()
						}
					}(i, p)
				}
			}
			wg.Wait()

			for i, p := range processor {
				if !logEntries[i].Completed && p.Required {
					crw.responseErrorObject.AssignField(p.Name, "required processor didn't run")
					return
				}
				switch p.Async {
				case true:
					crw.processorLogsResponseAsync[p.Name] = logEntries[i]
				case false:
					crw.processorLogsResponseSync[p.Name] = logEntries[i]
				}
			}

		})
	}
}

func ingressPathMiddlewareFunc(thisRouter router.Router, s *state) func(http.Handler) http.Handler {

	if s == nil {
		logger.Error("state is nil")
		return nil
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			crw, ok := w.(*GechologResponseWriter)
			if !ok {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				logger.Error("failed to cast ResponseWriter to GechologResponseWriter")
				return
			}

			prefix := thisRouter.Path
			ingressSubPath := strings.TrimPrefix(r.URL.Path, prefix)

			store.Store(&crw.requestObject, &crw.requestErrorObject, "gl_path", &prefix)
			store.Store(&crw.requestObject, &crw.requestErrorObject, "ingress_subpath", &ingressSubPath)
			store.Store(&crw.requestObject, &crw.requestErrorObject, "outbound_subpath", &ingressSubPath)

			next.ServeHTTP(crw, r)
		})
	}
}

func outboundInboundPathMiddlewareFunc(thisRouter router.Router, listOfRouters []router.Router, s *state) func(http.Handler) http.Handler {

	if len(listOfRouters) == 0 {
		logger.Error("listOfRouters is empty")
		return nil
	}

	if s == nil {
		logger.Error("state is nil")
		return nil
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			crw, ok := w.(*GechologResponseWriter)
			if !ok {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				logger.Error("failed to cast ResponseWriter to GechologResponseWriter")
				return
			}
			prefix := thisRouter.Path
			outboundRouter := thisRouter

			newGL_Path, err := crw.requestObject.GetField("gl_path")
			if err == nil { // else no change
				str := ""
				err := json.Unmarshal(newGL_Path, &str)
				if err != nil {
					crw.requestErrorObject.AssignField("gl_path", err.Error())
				}
				if string(str) != thisRouter.Path {
					// set  request.original_gl_path to the original value
					store.Store(&crw.requestObject, &crw.requestErrorObject, "original_gl_path", &prefix)
					for _, r := range listOfRouters {
						if r.Path == string(str) {
							// We found a match
							logger.Debug(
								"Set outbound router",
								slog.String("context", "ingress"),
								slog.String("path", r.Path),
							)

							outboundRouter = r
							break
						}
					}
				}
			}
			routerParsedURL, _ := url.Parse(outboundRouter.Outbound.Url)        // We trust this works since checks are made of the config
			endpointParsedURL, _ := url.Parse(outboundRouter.Outbound.Endpoint) // We trust this works since checks are made of the config
			endpointPath := endpointParsedURL.Path

			outboundSubPath := func() string {

				if endpointPath != "" && endpointPath != "/" {
					// Endpoint subpath overrides request subpath
					logger.Debug("Endpoint subpath overrides request subpath", slog.String("context", "ingress"), slog.String("endpointPath", endpointPath))
					return endpointPath
				}
				outboundSubPathRaw, err := crw.requestObject.GetField("outbound_subpath")
				if err != nil {
					return ""
				}

				var tmpOutboundSubPath string
				// We found it
				err = json.Unmarshal(outboundSubPathRaw, &tmpOutboundSubPath) // populate subpath parameters
				if err != nil {
					crw.requestErrorObject.AssignField("outbound_subpath", err.Error())
					return ""
				}
				return tmpOutboundSubPath
			}()

			crw.outboundURL.Scheme = routerParsedURL.Scheme
			crw.outboundURL.Host = routerParsedURL.Host
			crw.outboundURL.Path = path.Join(routerParsedURL.Path, outboundSubPath)
			if strings.HasSuffix(strings.Trim(routerParsedURL.Path+outboundSubPath, " "), "/") {
				crw.outboundURL.Path += "/"
			}

			store.Store(&crw.requestObject, &crw.requestErrorObject, "url_path", crw.outboundURL.String())

			next.ServeHTTP(crw, r)

			store.Store(&crw.responseObject, &crw.responseErrorObject, "gl_path", outboundRouter.Path)
		})
	}
}

func standardRequestFunc(myClient *http.Client) http.Handler {

	if myClient == nil {
		logger.Error("http.Client is empty")
		return nil
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		crw, ok := w.(*GechologResponseWriter)
		if !ok {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			logger.Error("failed to cast ResponseWriter to GechologResponseWriter")
			return
		}

		outboundRequest, err := http.NewRequest(r.Method, crw.outboundURL.String(), crw.outboundBody)
		if err != nil {
			crw.inboundBody.WriteString(`{"error":"internal server error"}`)
			crw.inboundStatusCode = http.StatusInternalServerError

			logger.Error("failed to create request", slog.Any("error", err))
			return
		}

		for key, values := range crw.outboundHeaders {
			for _, value := range values {
				outboundRequest.Header.Add(key, value)
			}
		}

		resp, err := myClient.Do(outboundRequest)
		if err != nil {
			crw.inboundBody.WriteString(`{"error":"failure making request"}`)

			// Check if it's a timeout
			if strings.Contains(err.Error(), "Client.Timeout") || strings.Contains(err.Error(), "context deadline exceeded") {
				// 504 Gateway Timeout
				crw.inboundStatusCode = http.StatusGatewayTimeout
				return
			}

			// Check for DNS resolution errors
			var dnsErr *net.DNSError
			if ok := errors.As(err, &dnsErr); ok {
				// 503 Service Unavailable
				crw.inboundStatusCode = http.StatusServiceUnavailable
				return
			}

			// Check if the connection was refused
			var opErr *net.OpError
			if ok := errors.As(err, &opErr); ok && opErr.Op == "dial" {
				// 502 Bad Gateway (Connection refused)
				crw.inboundStatusCode = http.StatusBadGateway
				return
			}

			// SSL/TLS errors
			if strings.Contains(err.Error(), "x509: certificate") || strings.Contains(err.Error(), "tls: handshake") {
				// 502 Bad Gateway (TLS/SSL Error)
				crw.inboundStatusCode = http.StatusBadGateway
				return
			}

			// Generic network errors
			if strings.Contains(err.Error(), "no such host") || strings.Contains(err.Error(), "network is unreachable") {
				// 503 Service Unavailable (Network Error)
				crw.inboundStatusCode = http.StatusServiceUnavailable
				return
			}

			crw.inboundStatusCode = http.StatusInternalServerError
			logger.Warn("failed to make request", slog.Any("error", err))
			return
		}
		defer resp.Body.Close()

		// Copy the response body to the buffer
		_, err = io.Copy(crw.inboundBody, resp.Body)
		if err != nil {
			crw.inboundBody.WriteString(`{"error":"internal server error"}`)
			crw.inboundStatusCode = http.StatusInternalServerError

			logger.Error("failed to read body", slog.Any("error", err))
			return
		}

		// Store the status code
		crw.inboundStatusCode = resp.StatusCode

		// Store the headers
		for key, values := range resp.Header {
			for _, value := range values {
				crw.inboundHeaders.Add(key, value)
			}
		}
	})
}

func echoRequestFunc() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		crw, ok := w.(*GechologResponseWriter)
		if !ok {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			logger.Error("failed to cast ResponseWriter to GechologResponseWriter")
			return
		}

		// Copy the response body to the buffer
		_, err := io.Copy(crw.inboundBody, crw.outboundBody)
		if err != nil {
			crw.inboundBody.WriteString(`{"error":"internal server error"}`)
			crw.inboundStatusCode = http.StatusInternalServerError
			logger.Error("failed to read body", slog.Any("error", err))
			return
		}
		crw.inboundStatusCode = http.StatusOK

		for key, values := range crw.outboundHeaders {
			for _, value := range values {
				crw.inboundHeaders.Add(key, value)
			}
		}
	})
}
