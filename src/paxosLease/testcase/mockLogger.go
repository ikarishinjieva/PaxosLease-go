package testcase

import (
	"fmt"
)

type mockLogger struct {
	lines []string
}

func newMockLogger() *mockLogger {
	ret := mockLogger{}
	ret.lines = make([]string, 0, 100)
	return &ret
}

func (l *mockLogger) Tracef(p string, args ...interface{}) {
	l.lines = append(l.lines, fmt.Sprintf(p, args...))
}
