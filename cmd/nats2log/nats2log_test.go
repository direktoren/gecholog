package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

/*
func TestSetupConfig(t *testing.T) {
	t.Run("When config file does not exist, returns error", func(t *testing.T) {
		args := []string{"-o", "testfiles/nats2log_config_test_noexist.json"}
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		err := setupConfig(fs, args)
		assert.Error(t, err)
		assert.Equal(t, "open testfiles/nats2log_config_test_noexist.json: no such file or directory", err.Error())

		globalConfig = nats2log_config{}
	})

	t.Run("When config file is invalid, returns error", func(t *testing.T) {
		args := []string{"-o", "testfiles/nats2log_config_test_invalid.json"}
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		err := setupConfig(fs, args)
		assert.Error(t, err)
		assert.Equal(t, "error validating configuration", err.Error())

		globalConfig = nats2log_config{}
	})

	t.Run("When config file is valid, returns nil", func(t *testing.T) {
		args := []string{"-o", "testfiles/nats2log_config_test_filewriter.json"}
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		err := setupConfig(fs, args)
		assert.NoError(t, err)

		// resets globalConfig, since being global will carry over to other tests
		globalConfig = nats2log_config{}
	})

	// as writers implementations are added, we can add more tests here
}*/

type mockWriter struct {
	responseChan chan []byte
	rejectFunc   func([]byte) bool
}

func (m mockWriter) validate() error {
	return nil
}

func (m *mockWriter) open() error {
	return nil
}

func (m *mockWriter) close() {
	return
}

func (m *mockWriter) write(data []byte) error {
	if m.rejectFunc(data) {
		return errors.New("rejected")
	}
	//go func() {
	m.responseChan <- data
	//}()

	return nil
}

func TestProcessMessage(t *testing.T) {
	args := []string{"-o", "../../config/nats2log_config.json"}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	setupConfig(fs, args)

	attempts := 10

	tests := []struct {
		name          string
		retries       int
		attemptValue  int
		messageBytes  []byte
		rejectFunc    func([]byte) bool
		expectedOrder []byte
	}{
		{
			name:          "Successful Message Processing",
			retries:       3,
			attemptValue:  0,
			messageBytes:  []byte{'1', '2', '3', '4', '5'},
			rejectFunc:    func(data []byte) bool { return false },
			expectedOrder: []byte{'1', '2', '3', '4', '5'},
		},
		{
			name:          "Successful Message Processing longer",
			retries:       3,
			attemptValue:  0,
			messageBytes:  []byte{'1', '2', '3', '4', '5', '6', '7', '8', '9', 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z'},
			rejectFunc:    func(data []byte) bool { return false },
			expectedOrder: []byte{'1', '2', '3', '4', '5', '6', '7', '8', '9', 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z'},
		},
		{
			name:         "Reject one message == 2",
			retries:      3,
			attemptValue: 1,
			messageBytes: []byte{'1', '2', '3', '4', '5'},
			rejectFunc: func(data []byte) bool {
				if data[0] == '2' && attempts > 0 {
					attempts--
					return true
				}
				return false
			},
			expectedOrder: []byte{'1', '3', '4', '5', '2'},
		},
		{
			name:         "Reject one message == 2 many attempts",
			retries:      30,
			attemptValue: 12,
			messageBytes: []byte{'1', '2', '3', '4', '5'},
			rejectFunc: func(data []byte) bool {
				if data[0] == '2' && attempts > 0 {
					attempts--
					return true
				}
				return false
			},
			expectedOrder: []byte{'1', '3', '4', '5', '2'},
		},
		{
			name:         "Discard one message == 2",
			retries:      1,
			attemptValue: 2,
			messageBytes: []byte{'1', '2', '3', '4', '5'},
			rejectFunc: func(data []byte) bool {
				if data[0] == '2' && attempts > 0 {
					attempts--
					return true
				}
				return false
			},
			expectedOrder: []byte{'1', '3', '4', '5'},
		},
		{
			name:         "Discard one message == 2,4",
			retries:      1,
			attemptValue: 4,
			messageBytes: []byte{'1', '2', '3', '4', '5'},
			rejectFunc: func(data []byte) bool {
				if (data[0] == '2' || data[0] == '4') && attempts > 0 {
					attempts--
					return true
				}
				return false
			},
			expectedOrder: []byte{'1', '3', '5'},
		},
		// Define more test cases for retries, errors, and context cancellation
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempts = tt.attemptValue
			ctx, cancel := context.WithTimeout(context.Background(), 3000*time.Millisecond)

			defer cancel()

			ch := make(chan message)
			go func() {
				for _, m := range tt.messageBytes {
					ch <- message{data: []byte{m}, retries: 0}
				}
			}()

			// Prepare
			responseChan := make(chan []byte)
			globalConfig.Retries = tt.retries
			globalConfig.RetryDelay = 200
			logLevel.Set(slog.LevelWarn)

			globalConfig.bW = &mockWriter{responseChan: responseChan, rejectFunc: tt.rejectFunc}

			go processMessage(ctx, ch)

			// You may need to wait for processing or use synchronization primitives
			// time.Sleep(time.Millisecond * 100) // Example wait

			for i := 0; i < len(tt.expectedOrder); i++ {
				select {
				case <-ctx.Done():
					t.Error("context cancelled before all messages were processed")
					return
				case response := <-responseChan:
					assert.Equal(t, tt.expectedOrder[i], response[0])
					break
				}
			}
			func() {
				for {
					if attempts <= 0 {
						return
					}
					time.Sleep(10 * time.Millisecond)
				}
			}()

			cancel()
			time.Sleep(time.Millisecond * 1000) // Give time for context to cancel
			close(ch)
		})
	}
}
