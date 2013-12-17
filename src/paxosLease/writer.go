package paxosLease

type Writer interface {
	SendPaxosMsg(ip string, data interface{}) error
	BroadcastPaxosMsg(data interface{}) error
}
