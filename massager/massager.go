package massager

import (
	"github.com/flood4life/dnser"
	"github.com/flood4life/dnser/config"
)

type Massager struct {
	Desired []config.Item
	Current []dnser.DNSRecord
}

func (m Massager) CalculateNeededActions() dnser.Actions {
	putActions := make([]dnser.DNSRecord, 0)
	delActions := make([]dnser.DNSRecord, 0)

	for _, cfg := range m.Desired {
		tree := transformIntoTree(cfg.Domain, m.Current)
		flatCurrent := make([]dnser.DNSRecord, 0)
		for _, node := range tree {
			flatCurrent = append(flatCurrent, flattenTree(cfg.Domain, node)...)
		}
		haveARecord := haveARecord(cfg.IP, cfg.Domain, m.Current)
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

	return dnser.Actions{
		PutActions:    putActions,
		DeleteActions: delActions,
	}
}

func haveARecord(ip config.IP, domain config.Domain, records []dnser.DNSRecord) *dnser.DNSRecord {
	shouldRecord := dnser.DNSRecord{
		Alias:  false,
		Name:   domain,
		Target: config.Domain(ip),
	}
	if record := findRecordByName(domain, records); record != nil && record.Equal(shouldRecord) {
		return record
	}
	return nil
}

func findPutActions(have, want []dnser.DNSRecord) []dnser.DNSRecord {
	actions := make([]dnser.DNSRecord, 0)
	for _, wantRecord := range want {
		haveRecord := findRecordByName(wantRecord.Name, have)
		if haveRecord == nil || !haveRecord.Equal(wantRecord) {
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

func flattenTree(parent config.Domain, node config.Node) []dnser.DNSRecord {
	if node.IsLeaf() {
		return []dnser.DNSRecord{{
			Alias:  true,
			Name:   node.Value,
			Target: parent,
		}}
	}

	records := make([]dnser.DNSRecord, 0)
	for _, child := range node.Children {
		records = append(records, flattenTree(node.Value, child)...)
	}

	return records
}
