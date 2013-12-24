package paxosLease

type Writer interface {
	SendPaxosMsg(fromIp string, toIp string, data interface{}) error
	BroadcastPaxosMsg(fromIp string, data interface{}) error
	GetNodeTotalCount() int
}
