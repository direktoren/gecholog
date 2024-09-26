package timer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimer(t *testing.T) {
	assert := assert.New(t)

	// Array of tests
	tests := []struct {
		name               string
		startTime          time.Time
		stopTime           time.Time
		expectedDurationMs int64
	}{
		{
			name:               "Test1",
			startTime:          time.Date(2023, 10, 14, 12, 0, 0, 0, time.UTC),
			stopTime:           time.Date(2023, 10, 14, 12, 0, 1, 0, time.UTC),
			expectedDurationMs: 1000,
		},
		{
			name:               "Test2",
			startTime:          time.Date(2023, 10, 14, 12, 0, 0, 0, time.UTC),
			stopTime:           time.Date(2023, 10, 14, 12, 0, 2, 500000000, time.UTC),
			expectedDurationMs: 2500,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var tmr Timer

			// Start the timer
			tmr.SetStart(test.startTime)

			// Stop the timer
			tmr.SetStop(test.stopTime)

			// Assert the start time
			assert.Equal(test.startTime, tmr.GetStart(), "Start times should be equal")

			// Assert the stop time
			assert.Equal(test.stopTime, tmr.GetStop(), "Stop times should be equal")

			// Assert the duration
			assert.Equal(test.expectedDurationMs, tmr.durationMilliseconds, "Durations should be equal")

			// Marshal the timer to JSON and ensure no error occurs
			_, err := tmr.MarshalJSON()
			assert.Nil(err, "Error should be nil when marshaling to JSON")

			var secondTmr Timer
			secondTmr.SetStartFromTimer(tmr)
			secondTmr.SetStopFromTimer(tmr)

			// Assert the start time
			assert.Equal(test.startTime, secondTmr.GetStart(), "Start times should be equal")

			// Assert the stop time
			assert.Equal(test.stopTime, secondTmr.GetStop(), "Stop times should be equal")

			// Assert the duration
			assert.Equal(test.expectedDurationMs, secondTmr.durationMilliseconds, "Durations should be equal")

		})
	}
}
