package paxosLease

import (
	"fmt"
	"math"
	"paxosLease/debug"
	"regexp"
	"strconv"
	"time"
)

type proposer struct {
	restartCounter       int
	proposeId            uint64
	leaseProposeId       uint64 /*for extend lease*/
	nodeIp               string
	writer               Writer
	totalNodeCount       int
	minMajority          int
	openNodeCount        int
	accpetNodeCount      int
	timeout              int
	isLeaseOwner         bool
	logger               Logger
	preparing            bool
	proposing            bool
	preparingTimeout     *tick
	proposingTimeout     *tick
	extendLeaseTimeout   *tick
	prepareResponseMutex chan bool
	proposeResponseMutex chan bool
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
	ret.prepareResponseMutex = make(chan bool, 1)
	ret.prepareResponseMutex <- true
	ret.proposeResponseMutex = make(chan bool, 1)
	ret.proposeResponseMutex <- true
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

func (p *proposer) startPreparing(isExtendLease bool) {
	p.logger.Tracef("node %v: start preparing", p.nodeIp)
	p.stopTicks()
	p.preparing = true
	p.proposing = false
	p.preparingTimeout = newTick(p.onPreparingTimeout).start(PREPARING_TIMEOUT)
	p.proposeId = p.nextProposeId(p.proposeId, isExtendLease)

	p.broadcastPrepareRequest()
}

func (p *proposer) stopTicks() {
	if nil != p.preparingTimeout {
		p.preparingTimeout.stop()
		p.preparingTimeout = nil
	}
	if nil != p.proposingTimeout {
		p.proposingTimeout.stop()
		p.proposingTimeout = nil
	}
	if nil != p.extendLeaseTimeout {
		p.extendLeaseTimeout.stop()
		p.extendLeaseTimeout = nil
	}
}

func (p *proposer) broadcastPrepareRequest() {
	p.logger.Tracef("node %v: broadcast PrepareRequest : proposeId=%v", p.nodeIp, p.proposeId)
	p.openNodeCount = 0
	request := newMessage("PrepareRequest", p.nodeIp)
	request.ProposeId = p.proposeId
	p.writer.BroadcastPaxosMsg(p.nodeIp, request)
}

func (p *proposer) nextProposeId(currentId uint64, isExtendLease bool) uint64 {
	var delta uint64 = 1
	if isExtendLease {
		delta = uint64(math.Ceil(float64(MAX_LEASED_TIME) / float64(PREPARING_TIMEOUT)))
	}
	left := (currentId >> (PROPOSE_ID_WIDTH_NODEID + PROPOSE_ID_WIDTH_RESTART_COUNTER)) + delta
	mid := p.restartCounter
	right := p.getNodeId()
	return uint64(left<<(PROPOSE_ID_WIDTH_NODEID+PROPOSE_ID_WIDTH_RESTART_COUNTER)) |
		uint64(mid<<PROPOSE_ID_WIDTH_NODEID) |
		uint64(right)
}

func (p *proposer) OnPrepareResponse(msg Message) {
	<-p.prepareResponseMutex
	defer func() { p.prepareResponseMutex <- true }()

	p.logger.Tracef("node %v: got PrepareResponse from %v : proposeId=%v, acceptedProposeId=%v", p.nodeIp, msg.SourceIp, msg.ProposeId, msg.AcceptedProposeId)

	if p.proposeId != msg.ProposeId || !p.preparing {
		p.logger.Tracef("node %v: ignore the PrepareResponse", p.nodeIp)
		return
	}
	if 0 == msg.AcceptedProposeId || p.leaseProposeId == msg.AcceptedProposeId {
		p.openNodeCount++
	}
	if p.openNodeCount < p.minMajority {
		return
	}

	p.preparing = false
	p.proposing = true

	p.startProposing()
}

func (p *proposer) OnPrepareReject(msg Message) {
	<-p.prepareResponseMutex
	defer func() { p.prepareResponseMutex <- true }()

	p.logger.Tracef("node %v: got PrepareReject from %v : proposeId=%v, acceptedProposeId=%v", p.nodeIp, msg.SourceIp, msg.ProposeId, msg.AcceptedProposeId)

	if p.proposeId != msg.ProposeId || !p.preparing {
		p.logger.Tracef("node %v: ignore the PrepareReject", p.nodeIp)
		return
	}

	bits := uint(PROPOSE_ID_WIDTH_NODEID + PROPOSE_ID_WIDTH_RESTART_COUNTER)
	if (msg.AcceptedProposeId >> bits) > (p.proposeId >> bits) {
		//p.proposeId = HIGH(msg.AcceptedProposeId) + LOW(p.proposeId)
		p.proposeId = (p.proposeId - ((p.proposeId >> bits) << bits)) + ((msg.AcceptedProposeId >> bits) << bits)
	}
}

func (p *proposer) startProposing() {
	p.logger.Tracef("node %v: got majority positive PrepareResponse", p.nodeIp)
	p.preparingTimeout.stop()
	p.timeout = MAX_LEASED_TIME
	p.proposingTimeout = newTick(p.onProposingTimeout).start(p.timeout)

	p.broadcastProposeRequest()
}

func (p *proposer) broadcastProposeRequest() {
	p.logger.Tracef("node %v: broadcast ProposeRequest : proposeId=%v,ProposeTimeout=%v", p.nodeIp, p.proposeId, p.timeout)
	p.accpetNodeCount = 0
	ret := newMessage("ProposeRequest", p.nodeIp)
	ret.ProposeId = p.proposeId
	ret.ProposeTimeout = p.timeout
	p.writer.BroadcastPaxosMsg(p.nodeIp, ret)
}

func (p *proposer) OnProposeResponse(msg Message) {
	<-p.proposeResponseMutex
	defer func() { p.proposeResponseMutex <- true }()

	p.logger.Tracef("node %v: got ProposeResponse from %v: proposeId=%v", p.nodeIp, msg.SourceIp, msg.ProposeId)
	if p.proposeId != msg.ProposeId || !p.proposing {
		p.logger.Tracef("node %v: ignore the ProposeResponse", p.nodeIp)
		return
	}
	p.accpetNodeCount++
	if p.accpetNodeCount < p.minMajority {
		return
	}
	p.preparing = false
	p.proposing = false

	p.becomeLeaseOwner()
}

func (p *proposer) becomeLeaseOwner() {
	p.leaseProposeId = p.proposeId
	p.isLeaseOwner = true
	proposeLeftTime := int(p.proposingTimeout.expireTime.Sub(time.Now()).Seconds())

	if proposeLeftTime >= 3 {
		delayMs := (proposeLeftTime - 3) * 1000
		p.extendLeaseTimeout = newTick(p.onExtendLeaseTimeout).start(delayMs)
	}
	p.logger.Tracef("node %v become lease owner", p.nodeIp)
}

func (p *proposer) onPreparingTimeout() {
	p.logger.Tracef("node %v preparing is timeout, restart prepraing", p.nodeIp)
	p.leaseProposeId = 0
	p.isLeaseOwner = false
	p.startPreparing(false)
}

func (p *proposer) onProposingTimeout() {
	p.logger.Tracef("node %v proposing is expired, restart prepraing", p.nodeIp)
	p.leaseProposeId = 0
	p.isLeaseOwner = false
	p.startPreparing(false)
}

func (p *proposer) onExtendLeaseTimeout() {
	if debug.HasCond(fmt.Sprintf("node %v disable lease extension", p.nodeIp)) {
		return
	}
	p.logger.Tracef("node %v extend its lease", p.nodeIp)
	p.startPreparing(true)
}

func (p *proposer) Start() {
	p.startPreparing(false)
}

func (p *proposer) Stop() {
	if nil != p.preparingTimeout {
		p.preparingTimeout.stop()
	}
	if nil != p.proposingTimeout {
		p.proposingTimeout.stop()
	}
}
