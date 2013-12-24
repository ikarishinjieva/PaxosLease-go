package testcase

import (
	"fmt"
	"paxosLease"
	"paxosLease/debug"
)

type mockWriter struct {
	nodes map[string]*paxosLease.Node
}

func newMockWriter() *mockWriter {
	return &mockWriter{}
}

func (m *mockWriter) SendPaxosMsg(fromIp string, toIp string, data interface{}) error {
	if msg, ok := data.(paxosLease.Message); !ok {
		return fmt.Errorf("data is not PaxosLease.Message : %v", data)
	} else {
		if nil == m.nodes[toIp] {
			return fmt.Errorf("No node for ip %v", toIp)
		}
		go m.send(fromIp, m.nodes[toIp], msg)
		return nil
	}
}

func (m *mockWriter) BroadcastPaxosMsg(fromIp string, data interface{}) error {
	if msg, ok := data.(paxosLease.Message); !ok {
		return fmt.Errorf("data is not PaxosLease.Message : %v", data)
	} else {
		for _, node := range m.nodes {
			go m.send(fromIp, node, msg)
		}
		return nil
	}
}

func (m *mockWriter) GetNodeTotalCount() int {
	return len(m.nodes)
}

func (m *mockWriter) send(fromIp string, node *paxosLease.Node, msg paxosLease.Message) error {
	if debug.HasCond(fmt.Sprintf("node %v is offline", node.Ip)) {
		return nil
	}
	if debug.HasCond("network brain-split") && fromIp != node.Ip &&
		!debug.HasCond(fmt.Sprintf("network brain-split: node %v can send msg to node %v", fromIp, node.Ip)) {
		return nil
	}
	node.ProcessMsg(msg)
	return nil
}
