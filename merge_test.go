package tiledinference

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMerge(t *testing.T) {
	// Create 5 boxes, where the first and last should be merged, and the rest kept separate

	boxes := []Rect{
		{Class: 0, X1: 0, Y1: 0, X2: 10, Y2: 10},
		{Class: 0, X1: 20, Y1: 20, X2: 30, Y2: 30},
		{Class: 0, X1: 40, Y1: 40, X2: 50, Y2: 50},
		{Class: 0, X1: 60, Y1: 60, X2: 70, Y2: 70},
		{Class: 1, X1: 0, Y1: 0, X2: 9, Y2: 11},
	}

	options := DefaultMergeOptions()

	// Allow merging of different classes
	options.MergeDifferentClasses = true
	merged := MergeBoxes(boxes, &options)
	require.Equal(t, 4, len(merged))
	require.Equal(t, [][]int{{0, 4}, {1}, {2}, {3}}, merged)

	// Disallow merging of different classes
	options.MergeDifferentClasses = false
	merged = MergeBoxes(boxes, &options)
	require.Equal(t, 5, len(merged))
	require.Equal(t, [][]int{{0}, {1}, {2}, {3}, {4}}, merged)
}
