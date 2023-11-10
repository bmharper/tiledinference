package tiledinference

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMerge(t *testing.T) {

	boxes := []Box{
		{Tile: 0, Class: 0, X1: 0, Y1: 0, X2: 10, Y2: 10}, // Merge
		{Tile: 1, Class: 0, X1: 20, Y1: 20, X2: 30, Y2: 30},
		{Tile: 2, Class: 0, X1: 40, Y1: 40, X2: 50, Y2: 50},
		{Tile: 3, Class: 0, X1: 60, Y1: 60, X2: 70, Y2: 70},
		{Tile: 4, Class: 1, X1: 0, Y1: 0, X2: 9, Y2: 11},  // Merge
		{Tile: 4, Class: 1, X1: 1, Y1: 1, X2: 11, Y2: 11}, // Don't merge, because same tile as previous
	}

	options := DefaultMergeOptions()

	// Allow merging of different classes
	options.MergeDifferentClasses = true
	merged := MergeBoxes(boxes, &options)
	require.Equal(t, 5, len(merged))
	require.Equal(t, [][]int{{0, 4}, {1}, {2}, {3}, {5}}, merged)

	// Disallow merging of different classes
	options.MergeDifferentClasses = false
	merged = MergeBoxes(boxes, &options)
	require.Equal(t, 6, len(merged))
	require.Equal(t, [][]int{{0}, {1}, {2}, {3}, {4}, {5}}, merged)
}
