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
	putActions := make([]dnser.PutAction, 0)
	delActions := make([]dnser.DeleteAction, 0)
	for _, cfg := range m.Desired {
		tree := transformIntoTree(cfg.Domain, m.Current)
		flatCurrent := make([]dnser.DNSRecord, 0)
		for _, node := range tree {
			for _, r := range flattenTree(cfg.Domain, node) {
				flatCurrent = append(flatCurrent, r)
			}
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
		for _, a := range findPutActions(flatCurrent, flatDesired) {
			putActions = append(putActions, a)
		}
		for _, a := range findDeleteActions(flatCurrent, flatDesired) {
			delActions = append(delActions, a)
		}
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

func findPutActions(have, want []dnser.DNSRecord) []dnser.PutAction {
	actions := make([]dnser.PutAction, 0)
	for _, wantRecord := range want {
		haveRecord := findRecordByName(wantRecord.Name, have)
		if haveRecord == nil || !haveRecord.Equal(wantRecord) {
			actions = append(actions, dnser.PutAction(wantRecord))
		}
	}

	return actions
}

func findDeleteActions(have, want []dnser.DNSRecord) []dnser.DeleteAction {
	actions := make([]dnser.DeleteAction, 0)
	for _, haveRecord := range have {
		wantRecord := findRecordByName(haveRecord.Name, want)
		if wantRecord == nil {
			actions = append(actions, dnser.DeleteAction{Name: haveRecord.Name})
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
	if len(records) == 0 {
		return nil
	}
	children := make([]config.Node, 0)
	for _, r := range records {
		if r.Target == parent {
			children = append(children, config.Node{
				Value:    r.Name,
				Children: transformIntoTree(r.Name, records),
			})
		}
	}

	return children
}

func flattenTree(parent config.Domain, node config.Node) []dnser.DNSRecord {
	if len(node.Children) == 0 {
		return []dnser.DNSRecord{{
			Alias:  true,
			Name:   node.Value,
			Target: parent,
		}}
	}

	records := make([]dnser.DNSRecord, 0)
	for _, child := range node.Children {
		childRecords := flattenTree(node.Value, child)
		for _, cr := range childRecords {
			records = append(records, cr)
		}
	}

	return records
}
