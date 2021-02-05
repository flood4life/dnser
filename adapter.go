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

// NewAliasRecord constructs an alias DNSRecord from input strings.
func NewAliasRecord(name, target string) DNSRecord {
	return DNSRecord{
		Alias:  true,
		Name:   config.Domain(name),
		Target: config.Domain(target),
	}
}

// NewRecord constructs a non-alias DNSRecord from input strings.
func NewRecord(name, target string) DNSRecord {
	return DNSRecord{
		Alias:  false,
		Name:   config.Domain(name),
		Target: config.Domain(target),
	}
}

// NameZone returns the domain zone of Record's Name.
func (r DNSRecord) NameZone() config.Domain {
	return extractZone(r.Name)
}

// TargetZone returns the domain zone of Record's Target.
func (r DNSRecord) TargetZone() config.Domain {
	return extractZone(r.Target)
}

func extractZone(domain config.Domain) config.Domain {
	fields := strings.FieldsFunc(string(domain), func(r rune) bool {
		return r == '.'
	})
	return config.Domain(strings.Join(fields[len(fields)-2:], ".") + ".")
}

// ActionType is the type of actions to be performed on the record: Upsert or Delete.
type ActionType string

// Available Action Types
const (
	Upsert ActionType = "UPSERT"
	Delete ActionType = "DELETE"
)

// Action combines the action type and the DNS record.
type Action struct {
	Type   ActionType
	Record DNSRecord
}

// Lister implements List.
type Lister interface {
	// List lists all currently existing DNS records that the adapter has access to.
	List(ctx context.Context) ([]DNSRecord, error)
}

// Processor implements Process.
type Processor interface {
	// Process performs the appropriate changes for each action.
	Process(ctx context.Context, actions []Action) error
}

// Adapter combines Lister and Processor.
type Adapter interface {
	Lister
	Processor
}
