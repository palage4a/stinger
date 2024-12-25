package stinger

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRRContainerNext(t *testing.T) {
	tcs := []struct {
		in []int
	}{
		{
			[]int{1, 2, 3},
		},
		{
			[]int{},
		},
		{
			[]int{1},
		},
	}

	for i, tc := range tcs {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			p := NewRRContainer(tc.in)
			for i := range tc.in {
				assert.Equal(t, p.Next(), tc.in[i])
			}
			if len(tc.in) > 0 {
				assert.Equal(t, p.Next(), tc.in[0])
			}
		})
	}
}

func TestShuffle(t *testing.T) {
	for i, tc := range []struct {
		in []int
	}{
		{[]int{2, 1, 3, 4, -1}},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			old := make([]int, len(tc.in))
			copy(old, tc.in)
			Shuffle(tc.in)
			assert.NotEqual(t, old, tc.in)
		})
	}
}

func TestMultiplySlice(t *testing.T) {
	for i, tc := range []struct {
		s   []int
		c   int
		out []int
	}{
		{[]int{1}, 1, []int{1}},
		{[]int{-1, 2, 0, 4, 3, 2}, 2, []int{-1, -1, 2, 2, 0, 0, 4, 4, 3, 3, 2, 2}},
		{[]int{}, 1, nil},
		{[]int{}, 0, nil},
		{[]int{1}, 0, nil},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			actual := MultiplySlice(tc.s, tc.c)
			assert.Equal(t, tc.out, actual)
		})
	}
}

func TestSplitSlice(t *testing.T) {
	for i, tc := range []struct {
		s   []int
		c   int
		out [][]int
	}{
		{[]int{1, 2, 3, 4}, 2, [][]int{{1, 2}, {3, 4}}},
		{[]int{1, 2, 3, 4}, 3, [][]int{{1, 2, 3}, {4}}},
		{[]int{1, 2, 3}, 3, [][]int{{1, 2, 3}}},
		{[]int{1, 2}, 1, [][]int{{1}, {2}}},
		{[]int{1, 2}, 3, [][]int{{1, 2}}},
		{[]int{}, 1, nil},
		{[]int{1, 2}, 0, [][]int{{1}, {2}}},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			actual := SplitSlice(tc.s, tc.c)
			assert.Equal(t, tc.out, actual)
		})
	}
}
