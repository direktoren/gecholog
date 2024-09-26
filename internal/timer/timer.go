package timer

import (
	"encoding/json"
	"fmt"
	"time"
)

// Data Structure for Timer
type Timer struct {
	start                time.Time
	stop                 time.Time
	durationMilliseconds int64
}

// Get the start time.Time object
func (t Timer) GetStart() time.Time {
	return t.start
}

// Ge the stop time.Time object
func (t Timer) GetStop() time.Time {
	return t.stop
}

// Ge the duration time.Time object
func (t Timer) GetDuration() int64 {
	if t.durationMilliseconds == 0 {
		return -1
	}
	return t.durationMilliseconds
}

// Manually set start time
func (t *Timer) SetStart(n time.Time) {
	t.start = n
}

// Copy start time form a Timer
func (t *Timer) SetStartFromTimer(tmr Timer) {
	t.SetStart(tmr.start)
}

// Start the clock
func (t *Timer) Start() {
	t.SetStart(time.Now())
}

// Manually set stop time
func (t *Timer) SetStop(n time.Time) {
	t.stop = n
	t.durationMilliseconds = t.stop.Sub(t.start).Milliseconds()
}

// Copy stop time from Timer
func (t *Timer) SetStopFromTimer(tmr Timer) {
	t.SetStop(tmr.stop)
}

// Stop the clock
func (t *Timer) Stop() {
	t.SetStop(time.Now())
}

// Stringifyer
func (t Timer) String() string {
	return fmt.Sprintf("Start: %s Stop: %s Duration: %d", t.start.String()[:19], t.stop.String()[:19], t.durationMilliseconds)
}

// Marshaller
func (t Timer) MarshalJSON() ([]byte, error) {
	structToMarshal := struct {
		Start    time.Time `json:"start"`
		Stop     time.Time `json:"stop"`
		Duration int64     `json:"duration"`
	}{
		Start:    t.start,
		Stop:     t.stop,
		Duration: t.durationMilliseconds,
	}

	return json.Marshal(structToMarshal)
}
