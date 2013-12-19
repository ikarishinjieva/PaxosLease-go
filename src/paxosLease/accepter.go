package paxosLease

type accepter struct {
	highestPromisedProposeId uint64
	acceptedProposeId        uint64
	acceptedProposeId2       uint64
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
		a.sendPrepareReject(msg.SourceIp, msg.ProposeId)
		return
	}
	a.highestPromisedProposeId = msg.ProposeId
	a.sendPrepareResponse(msg.SourceIp, msg.ProposeId)
	return
}

func (a *accepter) sendPrepareReject(ip string, proposeId uint64) {
	ret := newMessage("PrepareReject", a.nodeIp)
	ret.ProposeId = proposeId
	ret.AcceptedProposeId = a.acceptedProposeId2 //use acceptedProposeId2 because acceptedProposedId will be cleared after lease expired
	a.logger.Tracef("node %v: send PrepareReject : proposeId=%v, acceptedProposeId=%v", a.nodeIp, ret.ProposeId, ret.AcceptedProposeId)
	a.writer.SendPaxosMsg(a.nodeIp, ip, ret)
}

func (a *accepter) sendPrepareResponse(ip string, proposeId uint64) {
	ret := newMessage("PrepareResponse", a.nodeIp)
	ret.ProposeId = proposeId
	ret.AcceptedProposeId = a.acceptedProposeId //maybe 0
	a.logger.Tracef("node %v: send PrepareResponse : proposeId=%v, acceptedProposeId=%v", a.nodeIp, ret.ProposeId, ret.AcceptedProposeId)
	a.writer.SendPaxosMsg(a.nodeIp, ip, ret)
}

func (a *accepter) OnProposeRequest(msg Message) {
	a.logger.Tracef("node %v: got ProposeRequest : proposeId=%v", a.nodeIp, msg.ProposeId)
	if msg.ProposeId < a.highestPromisedProposeId {
		return
	}
	a.acceptedProposeId = msg.ProposeId
	a.acceptedProposeId2 = msg.ProposeId
	a.acceptedProposeTimeout = msg.ProposeTimeout
	a.proposingTimeout = newTick(a.onTimeout).start(a.acceptedProposeTimeout)

	a.sendProposeResponse(msg.SourceIp, msg.ProposeId)
	return
}

func (a *accepter) sendProposeResponse(ip string, proposeId uint64) {
	ret := newMessage("ProposeResponse", a.nodeIp)
	ret.ProposeId = proposeId
	a.logger.Tracef("node %v: send ProposeResponse : proposeId=%v", a.nodeIp, proposeId)
	a.writer.SendPaxosMsg(a.nodeIp, ip, ret)
}

func (a *accepter) onTimeout() {
	a.acceptedProposeId = 0
}

func (a *accepter) Stop() {
	if nil != a.proposingTimeout {
		a.proposingTimeout.stop()
	}
}

func (a *accepter) SetAcceptedProposeIdForTest(id uint64) {
	a.acceptedProposeId = 0 //lease expired
	a.acceptedProposeId2 = id
}

func (a *accepter) SetHighestPromisedProposeIdForTest(id uint64) {
	a.highestPromisedProposeId = id
}
