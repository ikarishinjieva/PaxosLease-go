package testcase

import (
	"errors"
	"fmt"
	"io/ioutil"
	"paxosLease"
	"paxosLease/debug"
	"strings"
)

type testcase struct {
	filePath string
	quitChan chan bool
	envs     map[string]string
	nodes    map[string]*paxosLease.Node
	logger   *mockLogger
	writer   *mockWriter
}

func NewTestcase(path string) *testcase {
	ret := testcase{}
	ret.filePath = path
	ret.quitChan = make(chan bool, 0)
	ret.envs = make(map[string]string)
	ret.nodes = make(map[string]*paxosLease.Node)
	ret.logger = newMockLogger()
	ret.writer = newMockWriter()
	return &ret
}

func (t *testcase) Start() (err error) {
	debug.ResetConds()
	bytes, err := ioutil.ReadFile(t.filePath)
	if nil != err {
		return
	}
	err = nil
	finally := false
	or := false
	var ors []chan error
	for _, line := range strings.Split(string(bytes), "\n") {
		if debug.HasCond("print test detail") {
			debug.TmpLogf(line + "\n")
		}
		//OR:
		if "OR:" == line {
			ors = make([]chan error, 0)
			or = true
			continue
		}
		if or {
			if strings.HasPrefix(line, "\t") {
				orChan := make(chan error, 0)
				go func() {
					orChan <- t.doAction(line)
				}()
				ors = append(ors, orChan)
				continue
			} else {
				or = false
				for _, orChan := range ors {
					err = <-orChan
					if nil == err {
						goto STMT
					}
				}
				err = fmt.Errorf("OR statment failed, the next statement is \"%v\"", line)
			}
		}
	STMT:
		if nil == err {
			if actionErr := t.doAction(line); nil != actionErr {
				msg := fmt.Sprintf("case %v failed @\"%v\" with error:\n\t%v", t.filePath, line, actionErr)
				err = errors.New(msg)
			}
		} else if finally {
			t.doAction(line)
		} else {
			if line == "finally:" {
				finally = true
			}
		}
	}
	return err
}

func (t *testcase) SetEnv(key string, value string) {
	t.envs[key] = value
}

func (t *testcase) introduceEnv(str string) string {
	ret := str
	for key, value := range t.envs {
		ret = strings.Replace(ret, "{"+key+"}", value, -1)
	}
	return ret
}

func (t *testcase) GetLogs() []string {
	return t.logger.all
}
