package itemcatalog

import (
	"strconv"

	"github.com/tinywasm/events"
)

type MockPublisher struct {
	Events []events.Event
}

func (m *MockPublisher) Publish(e events.Event) {
	m.Events = append(m.Events, e)
}

var _ events.Publisher = (*MockPublisher)(nil)

type mockIDGen struct {
	counter int
}

func (m *mockIDGen) NewID() string {
	m.counter++
	return "test-id-" + strconv.Itoa(m.counter)
}
