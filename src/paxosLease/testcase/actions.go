package testcase

import (
	"fmt"
	"paxosLease"
	"paxosLease/debug"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func (t *testcase) descToAction(desc string, reg string, prevErr error, fn func(matches []string) error) (err error) {
	if nil != prevErr {
		return prevErr
	}
	regex := regexp.MustCompile(reg)
	if regex.MatchString(desc) {
		matches := regex.FindStringSubmatch(desc)
		err = fn(matches)
	}
	return
}

func (t *testcase) doAction(desc string) (err error) {
	err = t.descToAction(desc, "create (\\d) nodes", err, func(matches []string) error {
		count, _ := strconv.Atoi(matches[1])
		return t.createNodes(count)
	})
	err = t.descToAction(desc, "node (\\d) propose", err, func(matches []string) error {
		id, _ := strconv.Atoi(matches[1])
		return t.propose(id)
	})
	err = t.descToAction(desc, "assert log \"(.*)\"", err, func(matches []string) error {
		return t.assertLog(matches[1])
	})
	err = t.descToAction(desc, "print logs", err, func(matches []string) error {
		return t.printLogs()
	})
	err = t.descToAction(desc, "sleep (\\d+)", err, func(matches []string) error {
		secs, _ := strconv.Atoi(matches[1])
		return t.sleep(secs)
	})
	err = t.descToAction(desc, "assert no log \"(.*)\"", err, func(matches []string) error {
		return t.assertNoLog(matches[1])
	})
	err = t.descToAction(desc, "stop all nodes", err, func(matches []string) error {
		return t.stopAllNodes()
	})
	err = t.descToAction(desc, "node (\\d) is offline", err, func(matches []string) error {
		id, _ := strconv.Atoi(matches[1])
		return t.nodeIsOffline(id)
	})
	err = t.descToAction(desc, "clear logs", err, func(matches []string) error {
		return t.clearLogs()
	})
	err = t.descToAction(desc, "print test detail", err, func(matches []string) error {
		return t.printTestDetail()
	})
	err = t.descToAction(desc, "all accepters have run for a while", err, func(matches []string) error {
		return t.allAccepterHaveRunForAWhile()
	})
	err = t.descToAction(desc, "node (\\d) disable lease extension", err, func(matches []string) error {
		id, _ := strconv.Atoi(matches[1])
		return t.disableLeaseExtension(id)
	})
	err = t.descToAction(desc, "network brain-split \\[([^\\]]+)\\] \\[([^\\]]+)\\]", err, func(matches []string) error {
		return t.networkBrainSplit([]string{matches[1], matches[2]})
	})
	return err
}

func (t *testcase) createNodes(count int) (err error) {
	for i := 1; i <= count; i++ {
		node := paxosLease.NewNode(strconv.Itoa(i), t.writer, count, t.logger)
		t.nodes[strconv.Itoa(i)] = node
	}
	t.writer.nodes = t.nodes
	return nil
}

func (t *testcase) stopAllNodes() (err error) {
	for _, node := range t.nodes {
		go node.Stop()
	}
	return nil
}

func (t *testcase) propose(id int) (err error) {
	node := t.nodes[strconv.Itoa(id)]
	go node.Proposer.Start()
	return nil
}

func (t *testcase) assertLog(expectLog string) (err error) {
	pass := make(chan bool, 0)
	quitChan := make(chan bool, 1)
	go func() {
		for {
			for _, line := range t.logger.lines {
				if strings.HasSuffix(line, expectLog) {
					pass <- true
					return
				}
			}
			select {
			case <-quitChan:
				return
			default:
			}
		}
	}()
	timeoutChan := make(chan bool, 1)
	go func() {
		time.Sleep(5 * time.Second)
		timeoutChan <- true
	}()
	select {
	case <-pass:
		return
	case <-timeoutChan:
		quitChan <- true
		return fmt.Errorf("No expected log found : %v", expectLog)
	}
}

func (t *testcase) printLogs() (err error) {
	fmt.Println(strings.Join(t.GetLogs(), "\n"))
	return nil
}

func (t *testcase) sleep(secs int) (err error) {
	time.Sleep(time.Duration(secs) * time.Second)
	return nil
}

func (t *testcase) assertNoLog(log string) (err error) {
	for _, line := range t.logger.lines {
		if strings.HasSuffix(line, log) {
			return fmt.Errorf("Found unexpacted log : %v", log)
		}
	}
	return nil
}

func (t *testcase) nodeIsOffline(id int) (err error) {
	debug.SetCond(fmt.Sprintf("node %v is offline", id))
	return nil
}

func (t *testcase) clearLogs() (err error) {
	t.logger.lines = make([]string, 0, 100)
	return nil
}

func (t *testcase) printTestDetail() (err error) {
	debug.SetCond("print test detail")
	return nil
}

func (t *testcase) allAccepterHaveRunForAWhile() (err error) {
	for _, node := range t.nodes {
		node.Accepter.SetAcceptedProposeIdForTest(10000 << (paxosLease.PROPOSE_ID_WIDTH_RESTART_COUNTER + paxosLease.PROPOSE_ID_WIDTH_NODEID))
		node.Accepter.SetHighestPromisedProposeIdForTest(10001 << (paxosLease.PROPOSE_ID_WIDTH_RESTART_COUNTER + paxosLease.PROPOSE_ID_WIDTH_NODEID))
	}
	return nil
}

func (t *testcase) disableLeaseExtension(id int) (err error) {
	debug.SetCond(fmt.Sprintf("node %v disable lease extension", id))
	return nil
}

func (t *testcase) networkBrainSplit(groups []string) (err error) {
	debug.SetCond("network brain-split")
	for _, group := range groups {
		nodes := strings.Split(group, ",")
		for _, a := range nodes {
			for _, b := range nodes {
				if a != b {
					debug.SetCond(fmt.Sprintf("node %v can send msg to node %v", a, b))
				}
			}
		}
	}
	return nil
}
