package paxosLease

import (
	"math"
	"regexp"
	"strconv"
)

type proposer struct {
	restartCounter   int
	proposeId        uint64
	nodeIp           string
	writer           Writer
	totalNodeCount   int
	minMajority      int
	openNodeCount    int
	accpetNodeCount  int
	timeout          int
	isLeaseOwner     bool
	logger           Logger
	preparing        bool
	proposing        bool
	preparingTimeout *tick
	proposingTimeout *tick
}

func newProposer(nodeIp string, writer Writer, totalNodeCount int, logger Logger) *proposer {
	ret := proposer{}
	ret.nodeIp = nodeIp
	ret.restartCounter = 0 //TODO init
	ret.proposeId =
		uint64(0<<(PROPOSE_ID_WIDTH_NODEID+PROPOSE_ID_WIDTH_RESTART_COUNTER)) |
			uint64(ret.restartCounter<<PROPOSE_ID_WIDTH_NODEID) |
			uint64(ret.getNodeId())
	ret.writer = writer
	ret.minMajority = int(math.Ceil(float64((totalNodeCount + 1) / 2)))
	if nil != logger {
		ret.logger = logger
	} else {
		ret.logger = &blackholeLogger{}
	}
	return &ret
}

func (p *proposer) getNodeId() int {
	//TODO
	ret, _ := strconv.Atoi(regexp.MustCompile("\\d$").FindString(p.nodeIp))
	return ret
}

func (p *proposer) startPreparing() {
	p.logger.Tracef("node %v: start preparing", p.nodeIp)
	p.preparingTimeout = newTick(p.onPreparingTimeout).start(PREPARING_TIMEOUT)
	p.preparing = true
	p.proposing = false
	p.openNodeCount = 0
	p.accpetNodeCount = 0
	p.isLeaseOwner = false
	p.proposeId = p.nextProposeId(p.proposeId)

	p.logger.Tracef("node %v: broadcast PrepareRequest : proposeId=%v", p.nodeIp, p.proposeId)
	request := newMessage("PrepareRequest", p.nodeIp)
	request.ProposeId = p.proposeId
	p.writer.BroadcastPaxosMsg(request)
}

func (p *proposer) nextProposeId(currentId uint64) uint64 {
	left := (currentId >> (PROPOSE_ID_WIDTH_NODEID + PROPOSE_ID_WIDTH_RESTART_COUNTER)) + 1
	mid := p.restartCounter
	right := p.getNodeId()
	return uint64(left<<(PROPOSE_ID_WIDTH_NODEID+PROPOSE_ID_WIDTH_RESTART_COUNTER)) |
		uint64(mid<<PROPOSE_ID_WIDTH_NODEID) |
		uint64(right)
}

func (p *proposer) OnPrepareResponse(msg Message) {
	p.logger.Tracef("node %v: got PrepareResponse from %v : proposeId=%v, acceptedProposeId=%v", p.nodeIp, msg.SourceIp, msg.ProposeId, msg.AcceptedProposeId)
	if p.proposeId != msg.ProposeId || !p.preparing {
		p.logger.Tracef("node %v: ignore the PrepareResponse", p.nodeIp)
		return
	}
	if 0 == msg.AcceptedProposeId {
		p.openNodeCount++
	}
	if p.openNodeCount < p.minMajority {
		return
	}

	//Start proposing
	p.logger.Tracef("node %v: got majority positive PrepareResponse", p.nodeIp)
	p.preparingTimeout.stop()
	p.preparing = false
	p.proposing = true
	p.timeout = MAX_LEASED_TIME
	p.proposingTimeout = newTick(p.onProposingTimeout).start(p.timeout)

	p.logger.Tracef("node %v: send ProposeRequest : proposeId=%v,ProposeTimeout=%v", p.nodeIp, msg.ProposeId, p.timeout)
	ret := newMessage("ProposeRequest", p.nodeIp)
	ret.ProposeId = p.proposeId
	ret.ProposeTimeout = p.timeout
	p.writer.BroadcastPaxosMsg(ret)
}

func (p *proposer) OnProposeResponse(msg Message) {
	p.logger.Tracef("node %v: got ProposeResponse from %v: proposeId=%v, acceptedProposeId=%v", p.nodeIp, msg.SourceIp, msg.ProposeId, msg.AcceptedProposeId)
	if p.proposeId != msg.ProposeId || !p.proposing {
		p.logger.Tracef("node %v: ignore the ProposeResponse", p.nodeIp)
		return
	}
	p.accpetNodeCount++
	if p.accpetNodeCount < p.minMajority {
		return
	}

	//Got propose
	p.isLeaseOwner = true
	p.preparing = false
	p.proposing = false
	p.logger.Tracef("node %v become lease owner", p.nodeIp)
}

func (p *proposer) onPreparingTimeout() {
	p.logger.Tracef("node %v preparing is timeout, restart prepraing", p.nodeIp)
	p.startPreparing()
}

func (p *proposer) onProposingTimeout() {
	p.logger.Tracef("node %v proposing is timeout, restart prepraing", p.nodeIp)
	p.startPreparing()
}

func (p *proposer) Start() {
	p.startPreparing()
}

func (p *proposer) Stop() {
	if nil != p.preparingTimeout {
		p.preparingTimeout.stop()
	}
	if nil != p.proposingTimeout {
		p.proposingTimeout.stop()
	}
}
