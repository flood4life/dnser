package massager

import (
	"reflect"
	"testing"

	"github.com/flood4life/dnser"
	"github.com/flood4life/dnser/config"
)

var config1 = []config.Item{{
	IP:     "127.0.0.1",
	Domain: "example.org.",
	Aliases: []config.Node{{
		Value: "foo.example.org.",
		Children: []config.Node{{
			Value:    "bar.example.org.",
			Children: nil,
		}, {
			Value:    "baz.example.org.",
			Children: nil,
		}},
	}, {
		Value:    "foobar.example.org.",
		Children: nil,
	}},
}}
var set1 = []dnser.DNSRecord{{
	Alias:  false,
	Name:   "example.org.",
	Target: "127.0.0.1",
}, {
	Alias:  false,
	Name:   "another.org.",
	Target: "127.0.0.1",
}, {
	Alias:  true,
	Name:   "foo.example.org.",
	Target: "example.org.",
}, {
	Alias:  true,
	Name:   "bar.foo.example.org.",
	Target: "foo.example.org.",
}}
var actions1 = []dnser.Action{{
	Type: dnser.Upsert,
	Record: dnser.DNSRecord{
		Alias:  true,
		Name:   "bar.example.org.",
		Target: "foo.example.org.",
	}}, {
	Type: dnser.Upsert,
	Record: dnser.DNSRecord{
		Alias:  true,
		Name:   "baz.example.org.",
		Target: "foo.example.org.",
	}}, {
	Type: dnser.Upsert,
	Record: dnser.DNSRecord{
		Alias:  true,
		Name:   "foobar.example.org.",
		Target: "example.org.",
	}}, {
	Type: dnser.Delete,
	Record: dnser.DNSRecord{
		Alias:  true,
		Name:   "bar.foo.example.org.",
		Target: "foo.example.org.",
	}},
}

var groupedActions1 = [][]dnser.Action{{
	{
		Type: dnser.Delete,
		Record: dnser.DNSRecord{
			Alias:  true,
			Name:   "bar.foo.example.org.",
			Target: "foo.example.org.",
		},
	},
	{
		Type: dnser.Upsert,
		Record: dnser.DNSRecord{
			Alias:  true,
			Name:   "bar.example.org.",
			Target: "foo.example.org.",
		},
	},
	{
		Type: dnser.Upsert,
		Record: dnser.DNSRecord{
			Alias:  true,
			Name:   "baz.example.org.",
			Target: "foo.example.org.",
		},
	},
	{
		Type: dnser.Upsert,
		Record: dnser.DNSRecord{
			Alias:  true,
			Name:   "foobar.example.org.",
			Target: "example.org.",
		},
	},
},
}

func TestMassager_CalculateNeededActions(t *testing.T) {
	type fields struct {
		Desired []config.Item
		Current []dnser.DNSRecord
	}
	tests := []struct {
		name   string
		fields fields
		want   [][]dnser.Action
	}{{
		name: "all good",
		fields: fields{
			Desired: config1,
			Current: set1,
		},
		want: groupedActions1,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Massager{
				Desired: tt.fields.Desired,
				Current: tt.fields.Current,
			}
			if got := m.CalculateNeededActions(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CalculateNeededActions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMassager_SplitDependentActions(t *testing.T) {
	type fields struct {
		Desired []config.Item
		Current []dnser.DNSRecord
	}
	type args struct {
		actions []dnser.Action
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   [][]dnser.Action
	}{{
		name: "all good",
		fields: fields{
			Desired: config1,
			Current: set1,
		},
		args: args{actions: actions1},
		want: groupedActions1,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Massager{
				Desired: tt.fields.Desired,
				Current: tt.fields.Current,
			}
			if got := m.splitDependentActions(tt.args.actions); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitDependentActions() = %v, want %v", got, tt.want)
			}
		})
	}
}
