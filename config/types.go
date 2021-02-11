package config

// IP represents an IPv4.
type IP string

// Domain represents a web domain.
type Domain string

// APIVersion represents the version of the dnser configuration.
type APIVersion int

// Supported API Versions.
const (
	One APIVersion = 1
)

// Config is a structure that contains the API Version and the config Items.
type Config struct {
	APIVersion APIVersion
	Config     []Item
}

// Item represents a configuration of IP, Domain and Domain's aliases.
type Item struct {
	IP      IP
	Domain  Domain
	Aliases []Node
}

// Node is a config tree node.
type Node struct {
	Value    Domain
	Children []Node
}

// IsLeaf returns whether the node does not have any children.
func (n Node) IsLeaf() bool {
	return len(n.Children) == 0
}
