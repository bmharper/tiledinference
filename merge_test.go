package tiledinference

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMerge(t *testing.T) {
	tiling := MakeTiling(100, 100, 20, 20, 5)
	t.Logf("Tiling %v x %v", tiling.NumX, tiling.NumY)
	t.Logf("%v", dumpTile(tiling, 0, 0))
	t.Logf("%v", dumpTile(tiling, 1, 0))
	t.Logf("%v", dumpTile(tiling, 2, 0))

	boxes := []Box{
		{Tile: tiling.MakeTileIndex(0, 0), Class: 0, Rect: Rect{X1: 0, Y1: 0, X2: 20, Y2: 10}}, // Merge
		{Tile: tiling.MakeTileIndex(1, 0), Class: 0, Rect: Rect{X1: 20, Y1: 20, X2: 30, Y2: 30}},
		{Tile: tiling.MakeTileIndex(2, 0), Class: 0, Rect: Rect{X1: 40, Y1: 40, X2: 50, Y2: 50}},
		{Tile: tiling.MakeTileIndex(3, 0), Class: 0, Rect: Rect{X1: 60, Y1: 60, X2: 70, Y2: 70}},
		{Tile: tiling.MakeTileIndex(1, 0), Class: 1, Rect: Rect{X1: 10, Y1: 0, X2: 30, Y2: 11}}, // Merge
	}

	options := DefaultMergeOptions()

	// Allow merging of different classes
	options.MergeDifferentClasses = true
	groups, mergedBoxes := MergeBoxes(tiling, boxes, &options)
	require.Equal(t, 4, len(groups))
	require.Equal(t, 4, len(mergedBoxes))
	require.Equal(t, [][]int{{0, 4}, {1}, {2}, {3}}, groups)
	require.Equal(t, Box{Tile: 0, Class: 0, Rect: Rect{X1: 0, Y1: 0, X2: 30, Y2: 11}}, mergedBoxes[0]) // Note how Y2 went from 10 to 11 after expansion

	// Disallow merging of different classes
	options.MergeDifferentClasses = false
	groups, mergedBoxes = MergeBoxes(tiling, boxes, &options)
	require.Equal(t, 5, len(groups))
	require.Equal(t, 5, len(mergedBoxes))
	require.Equal(t, [][]int{{0}, {1}, {2}, {3}, {4}}, groups)
}

// Verify that in the case where there are two possible merge candidates in the
// neighbouring tile, we pick the best match.
func TestPrioritizedMerge(t *testing.T) {
	tiling := MakeTiling(100, 100, 20, 20, 2)
	t.Logf("Tiling %v x %v", tiling.NumX, tiling.NumY)
	t.Logf("%v", dumpTile(tiling, 0, 0))
	t.Logf("%v", dumpTile(tiling, 1, 0))

	// Tile boundaries:
	// 0: 0-20
	// 1: 16-36

	for iter := 0; iter < 2; iter++ {

		boxes := []Box{
			{Tile: tiling.MakeTileIndex(0, 0), Class: 0, Rect: Rect{X1: 0, Y1: 0, X2: 20, Y2: 10}},  // This will be the origin of the merge
			{Tile: tiling.MakeTileIndex(1, 0), Class: 0, Rect: Rect{X1: 16, Y1: 0, X2: 30, Y2: 10}}, // We want to choose this one, because it overlaps more
			{Tile: tiling.MakeTileIndex(1, 0), Class: 0, Rect: Rect{X1: 19, Y1: 0, X2: 30, Y2: 10}}, // Not this one
		}

		if iter == 1 {
			// Swap the order of the boxes, so that we test both cases (i.e. best match being first or second)
			boxes[1], boxes[2] = boxes[2], boxes[1]
		}

		options := DefaultMergeOptions()

		groups, mergedBoxes := MergeBoxes(tiling, boxes, &options)
		require.Equal(t, 2, len(groups))
		if iter == 0 {
			require.Equal(t, [][]int{{0, 1}, {2}}, groups)
		} else {
			require.Equal(t, [][]int{{0, 2}, {1}}, groups)
		}
		require.Equal(t, Box{Tile: 0, Class: 0, Rect: Rect{X1: 0, Y1: 0, X2: 30, Y2: 10}}, mergedBoxes[0])
	}
}

func dumpTile(tiling Tiling, tx, ty int) string {
	tr := tiling.TileRect(tx, ty)
	return fmt.Sprintf("Tile %v (%v,%v): %v,%v,%v,%v", tiling.MakeTileIndex(tx, ty), tx, ty, tr.X1, tr.Y1, tr.X2, tr.Y2)
}
