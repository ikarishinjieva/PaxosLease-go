package paxosLease

type Logger interface {
	Tracef(p string, args ...interface{})
}

type blackholeLogger struct {
}

func (b *blackholeLogger) Tracef(p string, args ...interface{}) {

}
