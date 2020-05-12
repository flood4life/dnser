package config

type IP string
type Domain string
type APIVersion int

const (
	One APIVersion = 1
)

type Config struct {
	APIVersion APIVersion
	Config     []Item
}

type Item struct {
	IP      IP
	Domain  Domain
	Aliases Node
}

type Node struct {
	Value    Domain
	Children []Node
}

func (n Node) IsLeaf() bool {
	return len(n.Children) == 0
}
