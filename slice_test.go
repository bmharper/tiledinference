package tiledinference

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTilePositions(t *testing.T) {

	validateSplit := func(srcSize, nnSize, minPadding int, expectedTileStart []int, log bool) {
		spaceBetween, numTiles := ComputeTileSpacingAndCount(srcSize, nnSize, minPadding)
		actualTileStart := []int{}
		for i := 0; i < numTiles; i++ {
			actualTileStart = append(actualTileStart, OriginAt(i, spaceBetween))
		}
		//actualTileStart := SplitDimension(srcSize, nnSize, minPadding)
		if expectedTileStart != nil {
			require.Equal(t, expectedTileStart, actualTileStart)
		}
		for i := 1; i < len(actualTileStart); i++ {
			overlap := (actualTileStart[i-1] + nnSize) - actualTileStart[i]
			require.GreaterOrEqual(t, overlap, minPadding)
		}

		// first tile must be at zero
		require.Equal(t, 0, actualTileStart[0])

		// last tile must be precisely at edge of image, unless the nn is larger than the image
		if srcSize >= nnSize {
			require.Equal(t, srcSize-nnSize, actualTileStart[len(actualTileStart)-1])
		}

		if log {
			overlap := 0
			if len(actualTileStart) > 1 {
				overlap = (actualTileStart[0] + nnSize) - actualTileStart[1]
			}
			t.Logf("SplitDimension(%4d, %4d, %2d) = %6d    %v", srcSize, nnSize, minPadding, overlap, actualTileStart)
		}
	}

	validateSplit(10, 11, 2, []int{0}, true)
	validateSplit(10, 10, 0, []int{0}, true)
	validateSplit(10, 10, 3, []int{0}, true)
	validateSplit(10, 5, 0, []int{0, 5}, true)
	validateSplit(14, 6, 1, []int{0, 4, 8}, true)
	validateSplit(20, 6, 2, nil, true)
	validateSplit(1024, 640, 32, nil, true)
	validateSplit(1280, 640, 32, nil, true)

	for imgSize := 14; imgSize < 20; imgSize++ {
		for nnSize := 6; nnSize <= 14; nnSize++ {
			for minPad := 1; minPad <= 2; minPad++ {
				validateSplit(imgSize, nnSize, minPad, nil, false)
			}
		}
	}

}

func boxToString(x1, y1, x2, y2 int) string {
	return fmt.Sprintf("%v,%v,%v,%v", x1, y1, x2, y2)
}

func TestMisc(t *testing.T) {
	ti := MakeTiling(2688, 1560, 640, 480, 32)
	require.Equal(t, 5, ti.NumX)
	require.Equal(t, 4, ti.NumY)
	require.Equal(t, "0,0,640,480", boxToString(ti.TileBox(0, 0)))
	require.Equal(t, "512,0,1152,480", boxToString(ti.TileBox(1, 0)))
	require.Equal(t, "2048,1080,2688,1560", boxToString(ti.TileBox(4, 3)))
}
