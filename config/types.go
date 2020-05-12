package config

import "gopkg.in/yaml.v3"

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

type yamlConfig struct {
	APIVersion APIVersion `yaml:"apiVersion"`
	Config     []yamlItem `yaml:"config"`
}

type Item struct {
	IP      IP
	Domain  Domain
	Aliases Node
}

type yamlItem struct {
	IP      IP        `yaml:"ip"`
	Domain  string    `yaml:"domain"`
	Aliases yaml.Node `yaml:"aliases"`
}

type Node struct {
	Value    Domain
	Children []Node
}
