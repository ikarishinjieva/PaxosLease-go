package paxosLease

type Node struct {
	Accepter *accepter
	Proposer *proposer
	Ip       string
}

func NewNode(nodeIp string, writer Writer, logger Logger, paxosIdPersister PaxosIdPersister) *Node {
	ret := Node{}
	ret.Accepter = newAccepter(nodeIp, writer, logger)
	ret.Proposer = newProposer(nodeIp, writer, logger, paxosIdPersister)
	ret.Ip = nodeIp
	return &ret
}

func (n *Node) Stop() {
	n.Accepter.Stop()
	n.Proposer.Stop()
}

func (n *Node) Start() {
	n.Proposer.Start()
}

func (node *Node) ProcessMsg(msg Message) {
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
}

func (n *Node) GetPaxosId() uint64 {
	return n.Proposer.GetLeaseProposeId()
}
