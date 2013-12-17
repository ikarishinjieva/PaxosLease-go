package paxosLease

type accepter struct {
	highestPromisedProposeId uint64
	acceptedProposeId        uint64
	acceptedProposeTimeout   int
	writer                   Writer
	logger                   Logger
	nodeIp                   string
	proposingTimeout         *tick
}

func newAccepter(nodeIp string, writer Writer, logger Logger) *accepter {
	ret := accepter{}
	ret.nodeIp = nodeIp
	ret.writer = writer
	if nil != logger {
		ret.logger = logger
	} else {
		ret.logger = &blackholeLogger{}
	}
	return &ret
}

func (a *accepter) OnPrepareRequest(msg Message) {
	a.logger.Tracef("node %v: got PrepareRequest : proposeId=%v", a.nodeIp, msg.ProposeId)
	if msg.ProposeId < a.highestPromisedProposeId {
		return
	}
	a.highestPromisedProposeId = msg.ProposeId
	ret := newMessage("PrepareResponse", a.nodeIp)
	ret.ProposeId = msg.ProposeId
	ret.AcceptedProposeId = a.acceptedProposeId //maybe 0
	a.logger.Tracef("node %v: send PrepareResponse : proposeId=%v, acceptedProposeId=%v", a.nodeIp, msg.ProposeId, msg.AcceptedProposeId)
	a.writer.SendPaxosMsg(msg.SourceIp, ret)
	return
}

func (a *accepter) OnProposeRequest(msg Message) {
	a.logger.Tracef("node %v: got ProposeRequest : proposeId=%v", a.nodeIp, msg.ProposeId)
	if msg.ProposeId < a.highestPromisedProposeId {
		return
	}
	a.acceptedProposeId = msg.ProposeId
	a.acceptedProposeTimeout = msg.ProposeTimeout
	a.proposingTimeout = newTick(a.onTimeout).start(a.acceptedProposeTimeout)
	ret := newMessage("ProposeResponse", a.nodeIp)
	ret.ProposeId = msg.ProposeId
	a.logger.Tracef("node %v: send ProposeResponse : proposeId=%v", a.nodeIp, msg.ProposeId)
	a.writer.SendPaxosMsg(msg.SourceIp, ret)
	return
}

func (a *accepter) onTimeout() {
	a.acceptedProposeId = 0
}

func (a *accepter) Stop() {
	if nil != a.proposingTimeout {
		a.proposingTimeout.stop()
	}
}
