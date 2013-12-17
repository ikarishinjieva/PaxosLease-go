package paxosLease

type Node struct {
	Accepter *accepter
	Proposer *proposer
	Ip       string
}

func NewNode(nodeIp string, writer Writer, totalNodeCount int, logger Logger) *Node {
	ret := Node{}
	ret.Accepter = newAccepter(nodeIp, writer, logger)
	ret.Proposer = newProposer(nodeIp, writer, totalNodeCount, logger)
	ret.Ip = nodeIp
	return &ret
}

func (n *Node) Stop() {
	n.Accepter.Stop()
	n.Proposer.Stop()
}
