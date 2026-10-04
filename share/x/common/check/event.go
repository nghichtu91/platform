package check

import (
	"strings"
	"time"
)

type Event struct {
	Name   string
	Period time.Duration
}

type Events []Event

// String
func (e Events) String() string {
	if len(e) == 0 {
		return "没有事件"
	}
	var sb strings.Builder
	sb.WriteByte('\n')
	for _, event := range e {
		sb.WriteString("\t阶段: ")
		sb.WriteString(event.Name)
		sb.WriteString(", 耗时: ")
		sb.WriteString(event.Period.String())
		sb.WriteString("\n")
	}
	return sb.String()
}

type EventBuilder struct {
	start  time.Time
	events Events
}

func CreateEventsBuilder() EventBuilder {
	return EventBuilder{
		start:  time.Now(),
		events: make([]Event, 0, 4),
	}
}

func (b *EventBuilder) AddEvent(name string) {
	var now = time.Now()
	b.events = append(b.events, Event{
		Name:   name,
		Period: time.Since(b.start),
	})
	b.start = now
}

// Events returns the events that have been added to the builder.
func (b *EventBuilder) Events() Events {
	events := b.events
	b.events = nil
	return events
}
