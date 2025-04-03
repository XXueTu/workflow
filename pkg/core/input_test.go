package core

import (
	"reflect"
	"testing"
)

func TestParseNodeInputs(t *testing.T) {
	type args struct {
		inputs        []Inputs
		parentOutputs map[string]any
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]any
		wantErr bool
	}{
		{
			name: "成功解析输入",
			args: args{
				inputs: []Inputs{
					{
						Name:  "input1",
						Type:  []string{"string"},
						Value: Value{Content: Content{BlockID: "parent1", Name: "output1"}},
					},
					{
						Name:  "input2",
						Type:  []string{"integer"},
						Value: Value{Content: Content{BlockID: "parent2", Name: "output2"}},
					},
				},
				parentOutputs: map[string]any{
					"parent1": map[string]any{"output1": "testString"},
					"parent2": map[string]any{"output2": 123},
				},
			},
			want: map[string]any{
				"input1": "testString",
				"input2": 123,
			},
			wantErr: false,
		},
		{
			name: "父节点输出中缺少数据",
			args: args{
				inputs: []Inputs{
					{
						Name:  "input1",
						Type:  []string{"string"},
						Value: Value{Content: Content{BlockID: "parent1", Name: "output1"}},
					},
				},
				parentOutputs: map[string]any{
					// "parent1": map[string]any{"output1": "testString"}, // 注释掉以模拟缺少数据的情况
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "类型不匹配",
			args: args{
				inputs: []Inputs{
					{
						Name:  "input1",
						Type:  []string{"integer"},
						Value: Value{Content: Content{BlockID: "parent1", Name: "output1"}},
					},
				},
				parentOutputs: map[string]any{
					"parent1": map[string]any{"output1": "testString"},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseNodeInputs(tt.args.inputs, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseNodeInputs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseNodeInputs() = %v, want %v", got, tt.want)
			}
		})
	}
}
