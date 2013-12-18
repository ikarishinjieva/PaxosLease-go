package testcase

import (
	"fmt"
)

type mockLogger struct {
	lines []string
	all   []string
}

func newMockLogger() *mockLogger {
	ret := mockLogger{}
	ret.lines = make([]string, 0, 100)
	ret.all = make([]string, 0, 100)
	return &ret
}

func (l *mockLogger) Tracef(p string, args ...interface{}) {
	log := fmt.Sprintf(p, args...)
	// fmt.Println(log)
	l.lines = append(l.lines, log)
	l.all = append(l.all, log)
}
