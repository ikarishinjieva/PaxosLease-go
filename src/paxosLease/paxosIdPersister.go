package paxosLease

import (
	"fmt"
)

type PaxosIdPersister interface {
	Get() (uint64, error)
	Set(id uint64) error
}

type NullPaxosIdPersister struct {
}

func (n *NullPaxosIdPersister) Get() (uint64, error) {
	return 0, fmt.Errorf("Null persister")
}

func (n *NullPaxosIdPersister) Set(id uint64) error {
	return nil
}
