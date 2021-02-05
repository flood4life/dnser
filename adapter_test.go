package dnser

import (
	"testing"

	"github.com/flood4life/dnser/config"
)

func TestDNSRecord_NameTLD(t *testing.T) {
	type fields struct {
		Alias  bool
		Name   config.Domain
		Target config.Domain
	}
	tests := []struct {
		name   string
		fields fields
		want   config.Domain
	}{{
		name: "long domain",
		fields: fields{
			Name: "long.example.org.",
		},
		want: "example.org.",
	}, {
		name:   "TLD",
		fields: fields{Name: "example.org."},
		want:   "example.org.",
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := DNSRecord{
				Alias:  tt.fields.Alias,
				Name:   tt.fields.Name,
				Target: tt.fields.Target,
			}
			if got := r.NameZone(); got != tt.want {
				t.Errorf("NameZone() = %v, want %v", got, tt.want)
			}
		})
	}
}
