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

func (m *mockWriter) SendPaxosMsg(ip string, data interface{}) error {
	if msg, ok := data.(paxosLease.Message); !ok {
		return fmt.Errorf("data is not PaxosLease.Message : %v", data)
	} else {
		if nil == m.nodes[ip] {
			return fmt.Errorf("No node for ip %v", ip)
		}
		go m.send(m.nodes[ip], msg)
		return nil
	}
}

func (m *mockWriter) BroadcastPaxosMsg(data interface{}) error {
	if msg, ok := data.(paxosLease.Message); !ok {
		return fmt.Errorf("data is not PaxosLease.Message : %v", data)
	} else {
		for _, node := range m.nodes {
			go m.send(node, msg)
		}
		return nil
	}
}

func (m *mockWriter) send(node *paxosLease.Node, msg paxosLease.Message) error {
	if debug.HasCond(fmt.Sprintf("node %v is offline", node.Ip)) {
		return nil
	}
	switch msg.MsgType {
	case "PrepareRequest":
		go node.Accepter.OnPrepareRequest(msg)
	case "ProposeRequest":
		go node.Accepter.OnProposeRequest(msg)
	case "PrepareResponse":
		go node.Proposer.OnPrepareResponse(msg)
	case "ProposeResponse":
		go node.Proposer.OnProposeResponse(msg)
	case "PrepareReject":
		go node.Proposer.OnPrepareReject(msg)
	}
	return nil
}
