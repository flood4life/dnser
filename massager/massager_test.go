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
	Aliases: config.Node{
		Value: "example.org.",
		Children: []config.Node{{
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
	},
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
var actions1 = dnser.Actions{
	PutActions: []dnser.PutAction{{
		Alias:  true,
		Name:   "bar.example.org.",
		Target: "foo.example.org.",
	}, {
		Alias:  true,
		Name:   "baz.example.org.",
		Target: "foo.example.org.",
	}, {
		Alias:  true,
		Name:   "foobar.example.org.",
		Target: "example.org.",
	}},
	DeleteActions: []dnser.DeleteAction{{
		Name: "bar.foo.example.org.",
	}},
}

func TestMassager_CalculateNeededActions(t *testing.T) {
	type fields struct {
		Desired []config.Item
		Current []dnser.DNSRecord
	}
	tests := []struct {
		name   string
		fields fields
		want   dnser.Actions
	}{{
		name: "all good",
		fields: fields{
			Desired: config1,
			Current: set1,
		},
		want: actions1,
	},
	}
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
