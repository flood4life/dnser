package massager

import (
	"github.com/flood4life/dnser"
	"github.com/flood4life/dnser/config"
)

// Massager represents the difference between existing and desired set of DNS records.
type Massager struct {
	Desired []config.Item
	Current []dnser.DNSRecord
}

// CalculateNeededActions returns the list of actions necessary to transform
// the current state to the desired state.
func (m Massager) CalculateNeededActions() []dnser.Action {
	putActions := make([]dnser.DNSRecord, 0)
	delActions := make([]dnser.DNSRecord, 0)

	for _, cfg := range m.Desired {
		treeCurrent := transformIntoTree(cfg.Domain, m.Current)
		flatCurrent := flattenTree(cfg.Domain, treeCurrent)

		haveARecord := findPresentARecord(cfg.IP, cfg.Domain, m.Current)
		if haveARecord != nil {
			flatCurrent = append(flatCurrent, *haveARecord)
		}

		flatDesired := flattenTree(cfg.Domain, cfg.Aliases)
		wantARecord := dnser.DNSRecord{
			Alias:  false,
			Name:   cfg.Domain,
			Target: config.Domain(cfg.IP),
		}
		flatDesired = append(flatDesired, wantARecord)

		putActions = append(putActions, findPutActions(flatCurrent, flatDesired)...)
		delActions = append(delActions, findDeleteActions(flatCurrent, flatDesired)...)
	}

	return append(
		recordsToActions(putActions, dnser.Upsert),
		recordsToActions(delActions, dnser.Delete)...,
	)
}

// SplitDependentActions splits the flat list of dnser.Action into several lists.
// Actions inside a list may be executed concurrently, but the top-level lists
// need to be executed in the order they are presented, because records in list i+1
// reference records in list i.
func (m Massager) SplitDependentActions(actions []dnser.Action) [][]dnser.Action {
	// Traverse the tree with DFS and for each node check
	// if it needs to be upserted and the amount of predecessor
	// nodes (increment count on each jump)
	// then group by the amount of predecessor nodes

	// The first list is delete actions,
	// because they don't depend on anything else.
	groups := make(map[int][]dnser.Action)
	groups[0] = filterDeleteActions(actions)

	addAction := func(action dnser.Action, bucket int) {
		if groups[bucket] == nil {
			groups[bucket] = make([]dnser.Action, 0)
		}
		groups[bucket] = append(groups[bucket], action)
	}
	callback := func(domain config.Domain, dependentDomains int) bool {
		dependencyAction := findDomainUpsertAction(domain, actions)
		if dependencyAction == nil {
			return false
		}
		addAction(*dependencyAction, dependentDomains)
		return true
	}

	for _, root := range m.Desired {
		traverseTree(root, callback)
	}

	result := make([][]dnser.Action, len(groups))
	for i, group := range groups {
		result[i] = group
	}

	return result
}

func findDomainUpsertAction(domain config.Domain, actions []dnser.Action) *dnser.Action {
	for _, a := range actions {
		if a.Record.Name == domain {
			if a.Type == dnser.Upsert {
				return &a
			} else {
				return nil
			}
		}
	}
	return nil
}

func filterDeleteActions(actions []dnser.Action) []dnser.Action {
	result := make([]dnser.Action, 0)
	for _, a := range actions {
		if a.Type != dnser.Delete {
			continue
		}
		result = append(result, a)
	}
	return result
}

type treeCallback func(domain config.Domain, dependentDomains int) bool

func traverseTree(root config.Item, callback treeCallback) {
	dependentDomains := 0
	if absent := callback(root.Domain, dependentDomains); absent {
		dependentDomains++
	}

	for _, n := range root.Aliases {
		traverseNode(n, dependentDomains, callback)
	}
}

func traverseNode(node config.Node, dependentDomains int, callback treeCallback) {
	if absent := callback(node.Value, dependentDomains); absent {
		dependentDomains++
	}
	for _, n := range node.Children {
		traverseNode(n, dependentDomains, callback)
	}
}

func recordsToActions(records []dnser.DNSRecord, actionType dnser.ActionType) []dnser.Action {
	result := make([]dnser.Action, len(records))
	for i, r := range records {
		result[i] = dnser.Action{
			Type:   actionType,
			Record: r,
		}
	}
	return result
}

func findPresentARecord(ip config.IP, domain config.Domain, records []dnser.DNSRecord) *dnser.DNSRecord {
	shouldRecord := dnser.DNSRecord{
		Alias:  false,
		Name:   domain,
		Target: config.Domain(ip),
	}
	if record := findRecordByName(domain, records); record != nil && *record == shouldRecord {
		return record
	}
	return nil
}

func findPutActions(have, want []dnser.DNSRecord) []dnser.DNSRecord {
	actions := make([]dnser.DNSRecord, 0)
	for _, wantRecord := range want {
		haveRecord := findRecordByName(wantRecord.Name, have)
		if haveRecord == nil || !(*haveRecord == wantRecord) {
			actions = append(actions, wantRecord)
		}
	}

	return actions
}

func findDeleteActions(have, want []dnser.DNSRecord) []dnser.DNSRecord {
	actions := make([]dnser.DNSRecord, 0)
	for _, haveRecord := range have {
		wantRecord := findRecordByName(haveRecord.Name, want)
		if wantRecord == nil {
			actions = append(actions, haveRecord)
		}
	}

	return actions
}

func findRecordByName(name config.Domain, records []dnser.DNSRecord) *dnser.DNSRecord {
	for _, r := range records {
		if r.Name == name {
			return &r
		}
	}
	return nil
}

func transformIntoTree(parent config.Domain, records []dnser.DNSRecord) []config.Node {
	children := make([]config.Node, 0)
	for _, r := range records {
		if r.Target != parent {
			continue
		}
		children = append(children, config.Node{
			Value:    r.Name,
			Children: transformIntoTree(r.Name, records),
		})
	}

	return children
}

func flattenTree(parent config.Domain, nodes []config.Node) []dnser.DNSRecord {
	if len(nodes) == 0 {
		return nil
	}

	records := make([]dnser.DNSRecord, 0, len(nodes))
	for _, node := range nodes {
		records = append(records, dnser.DNSRecord{
			Alias:  true,
			Name:   node.Value,
			Target: parent,
		})
		records = append(records, flattenTree(node.Value, node.Children)...)
	}
	return records
}
