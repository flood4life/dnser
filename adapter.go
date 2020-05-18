package dnser

import (
	"context"
	"strings"

	"github.com/flood4life/dnser/config"
)

// DNSRecord contains the minimal set of data needed to represent an A DNS record.
type DNSRecord struct {
	Alias bool

	Name   config.Domain
	Target config.Domain
}

func (r DNSRecord) NameTLD() config.Domain {
	return extractTLD(r.Name)
}

func (r DNSRecord) TargetTLD() config.Domain {
	return extractTLD(r.Target)
}

func extractTLD(domain config.Domain) config.Domain {
	fields := strings.FieldsFunc(string(domain), func(r rune) bool {
		return r == '.'
	})
	return config.Domain(strings.Join(fields[len(fields)-2:], ".") + ".")
}

type ActionType string

const (
	Upsert ActionType = "UPSERT"
	Delete ActionType = "DELETE"
)

type Action struct {
	Type   ActionType
	Record DNSRecord
}

type Lister interface {
	// List lists all currently existing DNS records that the adapter has access to.
	List(ctx context.Context) ([]DNSRecord, error)
}

type Processor interface {
	// Process performs the appropriate changes for each action.
	Process(ctx context.Context, actions []Action) error
}

type Adapter interface {
	Lister
	Processor
}
