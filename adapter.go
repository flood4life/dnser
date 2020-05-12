package dnser

import (
	"context"

	"github.com/flood4life/dnser/config"
)

// DNSRecord contains the minimal set of data needed to represent an A DNS record.
type DNSRecord struct {
	Alias bool

	Name   config.Domain
	Target config.Domain
}

func (r DNSRecord) Equal(other DNSRecord) bool {
	return r.Alias == other.Alias && r.Name == other.Name && r.Target == other.Target
}

type Actions struct {
	PutActions    []DNSRecord
	DeleteActions []DNSRecord
}

type Lister interface {
	// List lists all currently existing DNS records that the adapter has access to.
	List(ctx context.Context) ([]DNSRecord, error)
}

type Processor interface {
	// Process performs the appropriate changes for each action.
	Process(ctx context.Context, actions Actions) error
}

type Adapter interface {
	Lister
	Processor
}
