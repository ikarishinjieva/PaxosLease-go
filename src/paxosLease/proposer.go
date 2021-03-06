package paxosLease

import (
	"fmt"
	"math"
	"math/rand"
	"paxosLease/debug"
	"regexp"
	"strconv"
	"time"
)

type proposer struct {
	restartCounter       int
	proposeId            uint64
	leaseProposeId       uint64 /* (leaseProposeId == 0) as isLeaseOwner */
	nodeIp               string
	writer               Writer
	openNodeCount        int
	accpetNodeCount      int
	leaseTime            int
	logger               Logger
	preparingTick        *tick
	proposingTick        *tick
	leaseTick            *tick
	extendLeaseTick      *tick
	giveupLeaseTick      *tick
	prepareResponseMutex chan bool
	proposeResponseMutex chan bool
	giveupLeaseMutex     chan bool
	stopped              bool
	proposeIdPersister   PaxosIdPersister
}

func newProposer(nodeIp string, writer Writer, logger Logger, proposeIdPersister PaxosIdPersister) *proposer {
	ret := proposer{}
	ret.nodeIp = nodeIp
	ret.restartCounter = 0 //TODO init
	if nil != logger {
		ret.logger = logger
	} else {
		ret.logger = &blackholeLogger{}
	}
	ret.proposeIdPersister = proposeIdPersister
	if id, err := proposeIdPersister.Get(); nil == err {
		ret.logger.Tracef("Proposer read persist propose id %v", id)
		ret.proposeId = id
	} else {
		ret.logger.Tracef("Proposer fail to read persist propose id, err is %v", err)
		ret.proposeId =
			uint64(0<<(PROPOSE_ID_WIDTH_NODEID+PROPOSE_ID_WIDTH_RESTART_COUNTER)) |
				uint64(ret.restartCounter<<PROPOSE_ID_WIDTH_NODEID) |
				uint64(ret.getNodeId())
	}
	ret.writer = writer
	ret.prepareResponseMutex = make(chan bool, 1)
	ret.prepareResponseMutex <- true
	ret.proposeResponseMutex = make(chan bool, 1)
	ret.proposeResponseMutex <- true
	ret.giveupLeaseMutex = make(chan bool, 1)
	ret.giveupLeaseMutex <- true
	return &ret
}

func (p *proposer) GetMinMajority() int {
	return int(math.Ceil((float64(p.writer.GetNodeTotalCount()) + 1) / float64(2)))
}

func (p *proposer) getNodeId() int {
	match := regexp.MustCompile("^(\\d+)\\.(\\d+)\\.(\\d+)\\.(\\d+)").FindStringSubmatch(p.nodeIp)
	if 0 == len(match) { //for unit test
		id, _ := strconv.Atoi(p.nodeIp)
		return id
	}
	seg4, _ := strconv.Atoi(match[4])
	seg3, _ := strconv.Atoi(match[3])
	port, _ := strconv.Atoi(regexp.MustCompile("\\d+$").FindString(p.nodeIp))
	id := (seg3+seg4)*10000 + port
	return id
}

func (p *proposer) startPreparing(isExtendLease bool) {
	if p.stopped {
		return
	}
	p.logger.Tracef("node %v: start preparing", p.nodeIp)
	if nil != p.giveupLeaseTick {
		p.logger.Tracef("node %v: ignore preparing, since it's giving up lease", p.nodeIp)
		return
	}
	p.stopTicks()
	p.preparingTick = newTick(p.onPreparingTimeout).start(PREPARING_TIMEOUT)
	p.proposeId = p.nextProposeId(p.proposeId, isExtendLease)

	p.broadcastPrepareRequest()
}

func (p *proposer) OnPrepareResponse(msg Message) {
	<-p.prepareResponseMutex
	defer func() { p.prepareResponseMutex <- true }()

	p.logger.Tracef("node %v: got PrepareResponse from %v : proposeId=%v, acceptedProposeId=%v", p.nodeIp, msg.SourceIp, msg.ProposeId, msg.AcceptedProposeId)

	if p.proposeId != msg.ProposeId || nil == p.preparingTick /*!p.preparing*/ {
		p.logger.Tracef("node %v: ignore the PrepareResponse", p.nodeIp)
		return
	}
	if 0 == msg.AcceptedProposeId || p.leaseProposeId == msg.AcceptedProposeId {
		p.openNodeCount++
	}
	if p.openNodeCount < p.GetMinMajority() {
		return
	}

	p.stopPreparingTick()
	p.startProposing()
}

func (p *proposer) startProposing() {
	p.logger.Tracef("node %v: got majority positive PrepareResponse", p.nodeIp)
	p.leaseTime = MAX_LEASED_TIME
	p.proposingTick = newTick(p.onProposingTimeout).start(p.leaseTime)

	p.broadcastProposeRequest()
}

func (p *proposer) OnPrepareReject(msg Message) {
	<-p.prepareResponseMutex
	defer func() { p.prepareResponseMutex <- true }()

	p.logger.Tracef("node %v: got PrepareReject from %v : proposeId=%v, acceptedProposeId=%v", p.nodeIp, msg.SourceIp, msg.ProposeId, msg.AcceptedProposeId)

	if p.proposeId != msg.ProposeId || nil == p.preparingTick /*!p.preparing*/ {
		p.logger.Tracef("node %v: ignore the PrepareReject", p.nodeIp)
		return
	}

	bits := uint(PROPOSE_ID_WIDTH_NODEID + PROPOSE_ID_WIDTH_RESTART_COUNTER)
	if (msg.AcceptedProposeId >> bits) > (p.proposeId >> bits) {
		//p.proposeId = HIGH(msg.AcceptedProposeId) + LOW(p.proposeId)
		p.proposeId = (p.proposeId - ((p.proposeId >> bits) << bits)) + ((msg.AcceptedProposeId >> bits) << bits)
	}
}

func (p *proposer) OnProposeResponse(msg Message) {
	<-p.proposeResponseMutex
	defer func() { p.proposeResponseMutex <- true }()

	p.logger.Tracef("node %v: got ProposeResponse from %v: proposeId=%v", p.nodeIp, msg.SourceIp, msg.ProposeId)
	if p.proposeId != msg.ProposeId || nil == p.proposingTick /*!p.proposing*/ {
		p.logger.Tracef("node %v: ignore the ProposeResponse", p.nodeIp)
		return
	}
	p.accpetNodeCount++
	if p.accpetNodeCount < p.GetMinMajority() {
		return
	}

	/*
		convert proposingTick to leaseTick
		DO NOT use p.stopProposingTick() here
	*/
	p.leaseTick = p.proposingTick
	p.proposingTick = nil

	p.becomeLeaseOwner()
}

func (p *proposer) becomeLeaseOwner() {
	p.leaseProposeId = p.proposeId
	leaseLeftTime := int(p.leaseTick.expireTime.Sub(time.Now()).Seconds())

	if leaseLeftTime >= 3 {
		delayMs := (leaseLeftTime - 3) * 1000
		p.extendLeaseTick = newTick(p.onExtendLeaseTimeout).start(delayMs)
	}
	p.logger.Tracef("node %v become lease owner", p.nodeIp)
}

func (p *proposer) onPreparingTimeout() {
	p.logger.Tracef("node %v preparing is timeout, restart prepraing", p.nodeIp)
	p.leaseProposeId = 0
	time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond) //solve dynamic dead-lock
	p.startPreparing(false)
}

func (p *proposer) onProposingTimeout() {
	p.logger.Tracef("node %v proposing is expired, restart prepraing", p.nodeIp)
	p.leaseProposeId = 0
	time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond) //solve dynamic dead-lock
	p.startPreparing(false)
}

func (p *proposer) IsLeaseOwner() bool {
	return 0 != p.leaseProposeId
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
	p.stopped = true
	p.stopTicks()
}

func (p *proposer) stopPreparingTick() {
	if nil != p.preparingTick {
		p.preparingTick.stop()
		p.preparingTick = nil
	}
}

func (p *proposer) stopProposingTick() {
	if nil != p.proposingTick {
		p.proposingTick.stop()
		p.proposingTick = nil
	}
}

func (p *proposer) stopLeaseTick() {
	if nil != p.leaseTick {
		p.leaseTick.stop()
		p.leaseTick = nil
	}
}

func (p *proposer) stopExtendLeaseTick() {
	if nil != p.extendLeaseTick {
		p.extendLeaseTick.stop()
		p.extendLeaseTick = nil
	}
}

func (p *proposer) stopGiveupLeaseTick() {
	if nil != p.extendLeaseTick {
		p.extendLeaseTick.stop()
		p.extendLeaseTick = nil
	}
}

func (p *proposer) stopTicks() {
	p.stopPreparingTick()
	p.stopProposingTick()
	p.stopExtendLeaseTick()
	p.stopLeaseTick()
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

func (p *proposer) broadcastProposeRequest() {
	p.logger.Tracef("node %v: broadcast ProposeRequest : proposeId=%v,ProposeTimeout=%v", p.nodeIp, p.proposeId, p.leaseTime)
	p.accpetNodeCount = 0
	ret := newMessage("ProposeRequest", p.nodeIp)
	ret.ProposeId = p.proposeId
	ret.ProposeTimeout = p.leaseTime
	p.writer.BroadcastPaxosMsg(p.nodeIp, ret)
}

func (p *proposer) GiveupLease() {
	<-p.giveupLeaseMutex
	defer func() { p.giveupLeaseMutex <- true }()
	if !p.IsLeaseOwner() {
		return
	}
	if nil != p.giveupLeaseTick {
		return
	}
	p.logger.Tracef("node %v request give up its lease", p.nodeIp)
	p.giveupLeaseTick = newTick(p.onGiveupLeaseTimeout).start(MAX_LEASED_TIME * 2)
	p.stopPreparingTick()
	p.stopExtendLeaseTick()
	//keep Proposing & Lease tick
}

func (p *proposer) onGiveupLeaseTimeout() {
	<-p.giveupLeaseMutex
	defer func() { p.giveupLeaseMutex <- true }()
	p.logger.Tracef("node %v give up its lease", p.nodeIp)
	p.giveupLeaseTick = nil
	p.leaseProposeId = 0
	p.startPreparing(false)
}

func (p *proposer) GetLeaseProposeId() uint64 {
	return p.leaseProposeId
}
