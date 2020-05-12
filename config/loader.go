package config

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadFromString creates a config from a string.
func LoadFromString(data string) (Config, error) {
	var yc yamlConfig
	if err := yaml.Unmarshal([]byte(data), &yc); err != nil {
		return Config{}, err
	}
	c := configFromYamlConfig(yc)
	return c, nil
}

type yamlConfig struct {
	APIVersion APIVersion `yaml:"apiVersion"`
	Config     []yamlItem `yaml:"config"`
}

type yamlItem struct {
	IP      IP        `yaml:"ip"`
	Domain  string    `yaml:"domain"`
	Aliases yaml.Node `yaml:"aliases"`
}

func configFromYamlConfig(yamlCfg yamlConfig) Config {
	cfg := Config{
		APIVersion: yamlCfg.APIVersion,
	}
	items := make([]Item, len(yamlCfg.Config))
	for i, cfgItem := range yamlCfg.Config {
		item := Item{
			IP:     cfgItem.IP,
			Domain: domainOfString(cfgItem.Domain),
		}
		item.Aliases = nodeFromYaml(cfgItem.Aliases)
		item.Aliases.Value = item.Domain
		items[i] = item
	}
	cfg.Config = items

	return cfg
}

func nodeFromYaml(node yaml.Node) Node {
	switch node.Kind {
	case yaml.ScalarNode:
		return makeLeafNode(node.Value)
	case yaml.SequenceNode:
		return makeInnerNode(node.Value, node.Content)
	case yaml.MappingNode:
		return makeInnerNode(node.Content[0].Value, node.Content[1].Content)
	default:
		return Node{}
	}
}

func makeInnerNode(value string, content []*yaml.Node) Node {
	children := make([]Node, len(content))
	for i, child := range content {
		children[i] = nodeFromYaml(*child)
	}
	return Node{
		Value:    domainOfString(value),
		Children: children,
	}
}

func makeLeafNode(value string) Node {
	return Node{
		Value:    domainOfString(value),
		Children: nil,
	}
}

func domainOfString(value string) Domain {
	if strings.HasSuffix(value, ".") {
		return Domain(value)
	}
	return Domain(value + ".")
}
