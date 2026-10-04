package check

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEvent(t *testing.T) {
	var builder = CreateEventsBuilder()
	time.Sleep(time.Millisecond * 10)
	builder.AddEvent("event1")
	runtime.Gosched()
	builder.AddEvent("event2")
	time.Sleep(time.Millisecond * 10)
	builder.AddEvent("event3")
	events := builder.Events()
	require.Equal(t, 3, len(events))
	t.Logf("events: %v", events)
}
