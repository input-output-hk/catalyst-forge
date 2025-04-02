package controller

import "time"

// MockClock is a Clock that can be controlled for testing.
type MockClock struct {
	CurrentTime time.Time
}

func (m *MockClock) Now() time.Time {
	return m.CurrentTime
}

func (m *MockClock) Since(t time.Time) time.Duration {
	return m.CurrentTime.Sub(t)
}

func (m *MockClock) Until(t time.Time) time.Duration {
	return t.Sub(m.CurrentTime)
}

func (m *MockClock) SetTime(t time.Time) {
	m.CurrentTime = t
}

func (m *MockClock) Advance(d time.Duration) {
	m.CurrentTime = m.CurrentTime.Add(d)
}

func NewMockClock(initialTime time.Time) *MockClock {
	return &MockClock{CurrentTime: initialTime}
}
