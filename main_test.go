package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_simulateObj(t *testing.T) {
	type args struct {
		serverState *obj
		actions     []clientInput
		curCycle    int
	}
	tests := []struct {
		name string
		args args
		want *obj
	}{
		{
			name: "just update",
			args: args{
				serverState: &obj{
					velocity: 1,
					pos:      0,
					cycle:    10,
				},
				curCycle: 20,
			},
			want: &obj{
				velocity: 1,
				pos:      9,
				cycle:    19,
			},
		},
		{
			name: "skip historic action",
			args: args{
				serverState: &obj{
					velocity: 1,
					pos:      0,
					cycle:    10,
				},
				actions: []clientInput{{
					cycle:    9,
					velocity: ptr(0.),
				}},
				curCycle: 20,
			},
			want: &obj{
				velocity: 1,
				pos:      9,
				cycle:    19,
			},
		},
		{
			name: "run a single action",
			args: args{
				serverState: &obj{
					velocity: 1,
					pos:      0,
					cycle:    10,
				},
				actions: []clientInput{{
					cycle:    11,
					velocity: ptr(0.),
				}},
				curCycle: 20,
			},
			want: &obj{
				velocity: 0.,
				pos:      0.,
				cycle:    19,
			},
		},
		{
			name: "run two actions",
			args: args{
				serverState: &obj{
					velocity: 1,
					pos:      0,
					cycle:    10,
				},
				actions: []clientInput{
					{
						cycle:    11,
						velocity: ptr(0.),
					},
					{
						cycle:    15,
						velocity: ptr(2.),
					},
				},
				curCycle: 20,
			},
			want: &obj{
				velocity: 2.,
				pos:      10.,
				cycle:    19,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := simulateObj(tt.args.serverState, tt.args.actions, tt.args.curCycle)
			assert.Equal(t, tt.want, got)
		})
	}
}
