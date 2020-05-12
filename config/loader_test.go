package config

import (
	"reflect"
	"testing"
)

const data1 = `apiVersion: 1
config:
- ip: 127.0.0.1
  domain: example.org
  aliases:
  - foo.example.org:
    - bar.example.org
    - baz.example.org
  - foobar.example.org 
`

var config1 = []Item{{
	IP:     "127.0.0.1",
	Domain: "example.org.",
	Aliases: Node{
		Value: "example.org.",
		Children: []Node{{
			Value: "foo.example.org.",
			Children: []Node{{
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

func TestLoadFromString(t *testing.T) {
	type args struct {
		data string
	}
	tests := []struct {
		name    string
		args    args
		want    Config
		wantErr bool
	}{
		{
			name: "all good",
			args: args{data: data1},
			want: Config{
				APIVersion: 1,
				Config:     config1,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadFromString(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LoadFromString() got = %v, want %v", got, tt.want)
			}
		})
	}
}
