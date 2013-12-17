package paxosLease

type Message struct {
	MsgType           string
	ProposeId         uint64
	AcceptedProposeId uint64
	SourceIp          string
	ProposeTimeout    int
}

func newMessage(msgType string, sourceIp string) Message {
	ret := Message{}
	ret.MsgType = msgType
	ret.SourceIp = sourceIp
	return ret
}
