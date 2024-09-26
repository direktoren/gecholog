package main

import (
	"bytes"
	"container/list"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"github.com/tidwall/gjson"
)

type logRecord struct {
	TransactionID string
	Router        string
	Time          string
	Latency       int
	StatusCode    int
	body          string
}
type threadLogList struct {
	listOfLogs  *list.List
	m           *sync.Mutex
	visibleLogs []logRecord
	length      int
}

func loggerSubscriberFunc(logList *threadLogList, natsEnabled *bool) (func(msg *nats.Msg), error) {
	if logList == nil {
		return nil, fmt.Errorf("logList is nil")
	}
	if logList.listOfLogs == nil {
		return nil, fmt.Errorf("logList.listOfLogs is nil")
	}
	if logList.m == nil {
		return nil, fmt.Errorf("logList.m is nil")
	}
	if natsEnabled == nil {
		return nil, fmt.Errorf("natsEnabled is nil")
	}
	return func(msg *nats.Msg) {
		logger.Debug(
			"receiver: received message",
			slog.String("message", string(msg.Data)),
		)
		logList.m.Lock()
		if logList.listOfLogs.Len() >= logList.length {
			last := logList.listOfLogs.Back()
			if last == nil {
				logger.Error("There is no last object. Abort", slog.Int("list length", logList.listOfLogs.Len()))
				*natsEnabled = false
				return
			}
			logList.listOfLogs.Remove(last)
			logger.Debug("removed object", slog.Int("list length", logList.listOfLogs.Len()))
		}
		logList.listOfLogs.PushFront(string(msg.Data))
		logger.Debug("Added to list", slog.Int("list length", logList.listOfLogs.Len()))
		logList.m.Unlock()
		// Pop, prepend to list
	}, nil
}

func logsListenerPageFunc(logList *threadLogList, natsEnabled *bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if logList == nil {
			logger.Error("logList is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}
		if logList.listOfLogs == nil {
			logger.Error("logList.listOfLogs is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}
		if logList.m == nil {
			logger.Error("logList.m is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}
		if natsEnabled == nil {
			logger.Error("natsEnabled is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if !*natsEnabled {
			c.Redirect(http.StatusFound, "mainmenu")
			logger.Debug("no nats connection. Redirect to mainmenu")
			return
		}

		noreload := c.Query("noreload")
		if noreload != "true" {
			// Let's reload the logs
			logList.m.Lock()
			logList.visibleLogs = make([]logRecord, 0, logList.listOfLogs.Len())
			for e := logList.listOfLogs.Front(); e != nil; e = e.Next() {
				val, ok := e.Value.(string)
				if !ok {
					logger.Error("invalid log loglist value", slog.Any("e.Value", e.Value))
					continue
				}
				transactionID := gjson.Get(val, "transaction_id")
				if transactionID.Type != gjson.String {
					logger.Error("error fetching transactionID", slog.Any("element", transactionID))
					continue
				}
				router := gjson.Get(val, "request.gl_path")
				if router.Type != gjson.String {
					logger.Error("error fetching router", slog.Any("element", router))
					continue
				}
				timeStamp := gjson.Get(val, "ingress_egress_timer.start")
				if timeStamp.Type != gjson.String {
					logger.Error("error fetching timeStamp", slog.Any("element", timeStamp))
					continue
				}
				latency := gjson.Get(val, "ingress_egress_timer.duration")
				// latency
				if latency.Type != gjson.Number {
					logger.Error("error fetching latency", slog.Any("element", latency))
					continue
				}
				statusCode := gjson.Get(val, "response.egress_status_code")
				if statusCode.Type != gjson.Number {
					logger.Error("error fetching statusCode", slog.Any("element", statusCode))
					continue
				}

				log := logRecord{
					TransactionID: transactionID.String(),
					Router:        router.String(),
					Time: func() string {
						t := timeStamp.String()
						if len(t) < 23 {
							return t
						}
						return t[:23]
					}(),
					Latency:    int(latency.Int()),
					StatusCode: int(statusCode.Int()),
					body:       val,
				}
				logList.visibleLogs = append(logList.visibleLogs, log)
			}
			logList.m.Unlock()
		}

		focusTransactionID := c.Query("transactionID")
		focusTransactionBody := ""

		if focusTransactionID != "" {
			logList.m.Lock()
			for _, log := range logList.visibleLogs {
				if log.TransactionID == focusTransactionID {
					var formattedJSON bytes.Buffer
					err := json.Indent(&formattedJSON, []byte(log.body), "", "   ")
					if err != nil {
						logger.Error("unable to format json", slog.Any("error", err))
						c.String(http.StatusInternalServerError, "Internal error")
						return
					}
					focusTransactionBody = formattedJSON.String()
					break
				}
			}
			logList.m.Unlock()
		}
		//logger.Debug("render!", slog.Any("Logs", logList.visibleLogs), slog.String("FocusID", focusTransactionID))
		c.HTML(http.StatusOK, "logs.html", gin.H{
			"Logs":    logList.visibleLogs,
			"FocusID": focusTransactionID,
			"Focus":   focusTransactionBody,
		})
	}
}
