package main

import (
	"container/list"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	sloggin "github.com/samber/slog-gin"
	"github.com/stretchr/testify/assert"
)

func Test_loggerSubscriberFunc_Bad_Setup(t *testing.T) {
	type args struct {
		logList     *threadLogList
		natsEnabled *bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "logList is nil",
			args: args{
				logList:     nil,
				natsEnabled: func() *bool { b := true; return &b }(),
			},
		},
		{
			name: "listOfLogs is nil",
			args: args{
				logList: &threadLogList{
					listOfLogs:  nil,
					m:           &sync.Mutex{},
					visibleLogs: []logRecord{},
				},
				natsEnabled: func() *bool { b := true; return &b }(),
			},
		},
		{
			name: "m is nil",
			args: args{
				logList: &threadLogList{
					listOfLogs:  &list.List{},
					m:           nil,
					visibleLogs: []logRecord{},
				},
				natsEnabled: func() *bool { b := true; return &b }(),
			},
		},
		{
			name: "natsEnabled is nil",
			args: args{
				logList: &threadLogList{
					listOfLogs:  &list.List{},
					m:           &sync.Mutex{},
					visibleLogs: []logRecord{},
				},
				natsEnabled: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := loggerSubscriberFunc(tt.args.logList, tt.args.natsEnabled)
			assert.Nil(t, got)
			assert.Error(t, err)

		})
	}
}

func Test_loggerSubscriberFunc_MsgHandling(t *testing.T) {
	logList := &threadLogList{
		listOfLogs:  &list.List{},
		m:           &sync.Mutex{},
		visibleLogs: []logRecord{},
		length:      5,
	}
	natsEnabled := true
	subscriber, err := loggerSubscriberFunc(logList, &natsEnabled)
	assert.NoError(t, err)
	tests := []struct {
		name                string
		messages            []string
		expextedListOrder   []string
		expectedListLength  int
		expectedNatsEnabled bool
	}{
		{
			name:                "Add 1 message",
			messages:            []string{"1"},
			expextedListOrder:   []string{"1"},
			expectedListLength:  1,
			expectedNatsEnabled: true,
		},
		{
			name:                "Add 5 messages",
			messages:            []string{"1", "2", "3", "4", "5"},
			expextedListOrder:   []string{"5", "4", "3", "2", "1"},
			expectedListLength:  5,
			expectedNatsEnabled: true,
		},
		{
			name:                "Add 6 messages",
			messages:            []string{"1", "2", "3", "4", "5", "6"},
			expextedListOrder:   []string{"6", "5", "4", "3", "2"},
			expectedListLength:  5,
			expectedNatsEnabled: true,
		},
		{
			name:                "Add 10 messages",
			messages:            []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"},
			expextedListOrder:   []string{"10", "9", "8", "7", "6"},
			expectedListLength:  5,
			expectedNatsEnabled: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, msg := range tt.messages {
				subscriber(&nats.Msg{Data: []byte(msg)})
			}

			assert.Equal(t, tt.expectedListLength, logList.listOfLogs.Len())

			for _, msg := range tt.expextedListOrder {
				assert.Equal(t, msg, logList.listOfLogs.Front().Value.(string))
				logList.listOfLogs.Remove(logList.listOfLogs.Front())
			}
			assert.Equal(t, tt.expectedNatsEnabled, natsEnabled)

		})
	}
}

func TestLogsListenerPageFunc_BadVars(t *testing.T) {

	// Define test scenarios
	tests := []struct {
		name        string
		logList     *threadLogList
		natsEnabled *bool
		queryParams map[string]string

		expectedResponseCode int
	}{
		{
			name: "basic success",
			logList: func() *threadLogList {
				t := &threadLogList{
					listOfLogs:  &list.List{},
					m:           &sync.Mutex{},
					visibleLogs: []logRecord{},
					length:      5,
				}
				t.listOfLogs.PushFront(`
				{
					"ingress_egress_timer": {
						"start": "2021-01-01T00:00:00Z",
						"duration": 45
					},
					"request": {
						"gl_path": "/test/"
					},
					"response": {
						"egress_status_code": 200
					},
					"transaction_id": "testKey"
				}`)
				return t
			}(),
			natsEnabled: func() *bool { b := true; return &b }(),
			queryParams: map[string]string{
				"transactionID": "testKey",
			},

			expectedResponseCode: http.StatusOK,
		},
		{
			name:        "loglist is nil",
			logList:     nil, // This is the error
			natsEnabled: func() *bool { b := true; return &b }(),
			queryParams: map[string]string{
				"key": "testKey",
			},

			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "logList.listOfLogs == nil",
			logList: func() *threadLogList {
				t := &threadLogList{
					listOfLogs:  nil, // This is the error
					m:           &sync.Mutex{},
					visibleLogs: []logRecord{},
					length:      5,
				}
				return t
			}(),
			natsEnabled: func() *bool { b := true; return &b }(),
			queryParams: map[string]string{
				"key": "testKey",
			},

			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "logList.m == nil",
			logList: func() *threadLogList {
				t := &threadLogList{
					listOfLogs:  &list.List{},
					m:           nil, // This is the error
					visibleLogs: []logRecord{},
					length:      5,
				}
				t.listOfLogs.PushFront(`
				{
					"ingress_egress_timer": {
						"start": "2021-01-01T00:00:00Z",
						"duration": 45
					},
					"request": {
						"gl_path": "/test/"
					},
					"response": {
						"egress_status_code": 200
					},
					"transaction_id": "testKey"
				}`)
				return t
			}(),
			natsEnabled: func() *bool { b := true; return &b }(),
			queryParams: map[string]string{
				"key": "testKey",
			},

			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "natsEnabled == nil",
			logList: func() *threadLogList {
				t := &threadLogList{
					listOfLogs:  &list.List{},
					m:           &sync.Mutex{},
					visibleLogs: []logRecord{},
					length:      5,
				}
				t.listOfLogs.PushFront(`
				{
					"ingress_egress_timer": {
						"start": "2021-01-01T00:00:00Z",
						"duration": 45
					},
					"request": {
						"gl_path": "/test/"
					},
					"response": {
						"egress_status_code": 200
					},
					"transaction_id": "testKey"
				}`)
				return t
			}(),
			natsEnabled: nil, // This is the error
			queryParams: map[string]string{
				"key": "testKey",
			},

			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "redirect",
			logList: func() *threadLogList {
				t := &threadLogList{
					listOfLogs:  &list.List{},
					m:           &sync.Mutex{},
					visibleLogs: []logRecord{},
					length:      5,
				}
				t.listOfLogs.PushFront(`
				{
					"ingress_egress_timer": {
						"start": "2021-01-01T00:00:00Z",
						"duration": 45
					},
					"request": {
						"gl_path": "/test/"
					},
					"response": {
						"egress_status_code": 200
					},
					"transaction_id": "testKey"
				}`)
				return t
			}(),
			natsEnabled: func() *bool { b := false; /* This is the error return */ return &b }(),
			queryParams: map[string]string{
				"key": "testKey",
			},

			expectedResponseCode: http.StatusFound,
		},
		{
			name: "bad json",
			logList: func() *threadLogList {
				t := &threadLogList{
					listOfLogs:  &list.List{},
					m:           &sync.Mutex{},
					visibleLogs: []logRecord{},
					length:      5,
				}
				t.listOfLogs.PushFront(`
				{
					"ingress_egress_timer": {
						"start": "2021-01-01T00:00:00Z",
						"duration": 45
					},
					"request": {
						"gl_path": "/test/"
					},
					"response": {
						"egress_status_code": 200
					},
					"transaction_id": "testKey"
				`) // This is the error
				return t
			}(),
			natsEnabled: func() *bool { b := true; return &b }(),
			queryParams: map[string]string{
				"transactionID": "testKey",
			},

			expectedResponseCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions

			gin.SetMode(gin.ReleaseMode)
			ginRouter := gin.New()
			ginRouter.Use(sloggin.New(logger))
			ginRouter.Use(gin.Recovery())
			ginRouter.LoadHTMLGlob("templates/logs.html")

			testFunc := logsListenerPageFunc(tt.logList, tt.natsEnabled)

			ginRouter.GET("/test", testFunc)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()

			ginRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedResponseCode, w.Code)

		})
	}
}

func TestLogsListenerPageFunc_Reload(t *testing.T) {

	// Define test scenarios
	tests := []struct {
		name        string
		logList     *threadLogList
		natsEnabled *bool
		queryParams map[string]string

		expectedVisibleLogs  []logRecord
		expectedResponseCode int
	}{
		{
			name: "reload",
			logList: func() *threadLogList {
				t := &threadLogList{
					listOfLogs: &list.List{},
					m:          &sync.Mutex{},
					visibleLogs: []logRecord{
						{
							TransactionID: "testKey0",
						},
						{
							TransactionID: "testKey1",
						},
						{
							TransactionID: "testKey2",
						},
					},
					length: 5,
				}
				t.listOfLogs.PushFront(`
				{
					"ingress_egress_timer": {
						"start": "2021-01-01T00:00:00Z",
						"duration": 45
					},
					"request": {
						"gl_path": "/test/"
					},					
					"response": {
						"egress_status_code": 200
					},
					"transaction_id": "testKey3"
				}`)
				t.listOfLogs.PushFront(`
				{
					"ingress_egress_timer": {
						"start": "2021-01-01T00:00:00Z",
						"duration": 45
					},
					"request": {
						"gl_path": "/test/"
					},
					"response": {
						"egress_status_code": 200
					},
					"transaction_id": "testKey4"
				}`)
				t.listOfLogs.PushFront(`
				{
					"ingress_egress_timer": {
						"start": "2021-01-01T00:00:00Z",
						"duration": 45
					},
					"request": {
						"gl_path": "/test/"
					},
					"response": {
						"egress_status_code": 200
					},
					"transaction_id": "testKey5"
				}`)
				return t
			}(),
			natsEnabled: func() *bool { b := true; return &b }(),
			queryParams: map[string]string{
				"noreload": "false",
			},

			expectedVisibleLogs: []logRecord{
				{
					TransactionID: "testKey5",
				},
				{
					TransactionID: "testKey4",
				},
				{
					TransactionID: "testKey3",
				},
			},
			expectedResponseCode: http.StatusOK,
		},
		{
			name: "no reload",
			logList: func() *threadLogList {
				t := &threadLogList{
					listOfLogs: &list.List{},
					m:          &sync.Mutex{},
					visibleLogs: []logRecord{
						{
							TransactionID: "testKey0",
						},
						{
							TransactionID: "testKey1",
						},
						{
							TransactionID: "testKey2",
						},
					},
					length: 5,
				}
				t.listOfLogs.PushFront(`
				{
					"ingress_egress_timer": {
						"start": "2021-01-01T00:00:00Z",
						"duration": 45
					},
					"request": {
						"gl_path": "/test/"
					},
					"response": {
						"egress_status_code": 200
					},
					"transaction_id": "testKey3"
				}`)
				t.listOfLogs.PushFront(`
				{
					"ingress_egress_timer": {
						"start": "2021-01-01T00:00:00Z",
						"duration": 45
					},
					"request": {
						"gl_path": "/test/"
					},
					"response": {
						"egress_status_code": 200
					},
					"transaction_id": "testKey4"
				}`)
				t.listOfLogs.PushFront(`
				{
					"ingress_egress_timer": {
						"start": "2021-01-01T00:00:00Z",
						"duration": 45
					},
					"request": {
						"gl_path": "/test/"
					},
					"response": {
						"egress_status_code": 200
					},
					"transaction_id": "testKey5"
				}`)
				return t
			}(),
			natsEnabled: func() *bool { b := true; return &b }(),
			queryParams: map[string]string{
				"noreload": "true",
			},

			expectedVisibleLogs: []logRecord{
				{
					TransactionID: "testKey0",
				},
				{
					TransactionID: "testKey1",
				},
				{
					TransactionID: "testKey2",
				},
			},
			expectedResponseCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions

			gin.SetMode(gin.ReleaseMode)
			ginRouter := gin.New()
			ginRouter.Use(sloggin.New(logger))
			ginRouter.Use(gin.Recovery())
			ginRouter.LoadHTMLGlob("templates/logs.html")

			testFunc := logsListenerPageFunc(tt.logList, tt.natsEnabled)

			ginRouter.GET("/test", testFunc)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()

			ginRouter.ServeHTTP(w, req)

			assert.Equal(t, len(tt.expectedVisibleLogs), len(tt.logList.visibleLogs))
			if len(tt.expectedVisibleLogs) == len(tt.logList.visibleLogs) {
				for i, log := range tt.expectedVisibleLogs {
					assert.Equal(t, log.TransactionID, tt.logList.visibleLogs[i].TransactionID)
				}
			}

			assert.Equal(t, tt.expectedResponseCode, w.Code)

		})
	}
}
