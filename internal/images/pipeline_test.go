package images

import (
	"reflect"
	"testing"
)

func TestChooseWidths(t *testing.T) {
	widths := []int{400, 800, 1200, 1600}
	cases := []struct {
		srcW int
		want []int
	}{
		{srcW: 200, want: []int{200}},
		{srcW: 315, want: []int{315}},
		{srcW: 400, want: []int{400}},
		{srcW: 520, want: []int{400, 520}},
		{srcW: 800, want: []int{400, 800}},
		{srcW: 1200, want: []int{400, 800, 1200}},
		{srcW: 1600, want: []int{400, 800, 1200, 1600}},
		{srcW: 1765, want: []int{400, 800, 1200, 1600, 1765}},
		{srcW: 2400, want: []int{400, 800, 1200, 1600, 2400}},
		{srcW: 3000, want: []int{400, 800, 1200, 1600, 2400}}, // capped
		{srcW: 6000, want: []int{400, 800, 1200, 1600, 2400}}, // capped
	}
	for _, c := range cases {
		got := chooseWidths(widths, c.srcW)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("srcW=%d: got %v, want %v", c.srcW, got, c.want)
		}
	}
}
