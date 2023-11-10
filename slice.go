package tiledinference

/*
Package tiledinference helps you split an image up into tiles for neural network inference.

The exact semantics of where to place the tiles can be somewhat tricky if
you've never encountered this problem before, but it's really quite simple
once you've figured out the principles.
The first thing to note is that we don't add padding on the outer edges of
the outer tiles, because that would imply running the neural network outside
of the image, which doesn't make any sense. We only add padding on the inside.
The following discussion talks about the horizontal (width) dimension only,
but the exact same logic applies to the vertical dimension too.

Firstly, we separate the discussion into the treatment of the exterior tiles
and the interior tiles. The exterior tiles have padding on one side only. Let's
call the width of the neural network "NN". When we talk about the "valid"
portion of a tile, it is the portion that is not part of the padding.
The entire input image is part of the "valid" region, and we need to consume
all of it with valid regions of tiles.
The valid size of the exterior tiles is (NN - Padding), because they lose padding
on the inside only. The valid size of the interior tiles is (NN - Padding * 2), because
they lose padding on both sides. By definition if we are tiling, then we
have at least two tiles, so this simplifies our calculations. Those minimum two
tiles are our exterior tiles, and we obviously then have an arbitrary number of
interior tiles (could be zero).

So let's figure out how many interior tiles we need.

	ExteriorValid = NN - Padding
	InteriorValid = NN - Padding * 2

We start with the whole image size, and remove the valid portion of the two outer tiles:

	InnerValid = ImageWidth - 2 * ExteriorValid

InnerValid is number of valid pixels that need to be covered by interior tiles.
Computing the number of inner tiles is trivial:

	InnerTiles = ceil(InnerValid / InteriorValid)

And the total number of tiles is likewise trivial:

	TotalTiles = 2 + InnerTiles

Now that we know TotalTiles, we can distribute them so that they are spread
evenly across the image. A naive solution might be to add tiles from the left
and then possibly end up with a thin sliver tile on the right. This is not good,
because the NN needs context, and this would imply running the NN beyond the
borders of the image. So what we do instead, is to first compute the number
of inner tiles needed, and then distribute them evenly within the image.
This means that in practice the padding is often quite a bit more than our
"minPadding" constant. This is why we name the parameter "minPadding".
Due to rounding, we can actually end up with 1-pixel padding differences
between different tiles.

*/

// Tiling defines how an image has been split into tiles
type Tiling struct {
	SpaceX      float64 // Horizontal pixels between each tile
	SpaceY      float64 // Vertical pixels between each tile
	NumX        int     // Number of tiles horizontally
	NumY        int     // Number of tiles vertically
	NNWidth     int     // Width of neural network
	NNHeight    int     // Height of neural network
	ImageWidth  int     // Width of original image
	ImageHeight int     // Height of original image
}

// Split an image up into tiles
func MakeTiling(imageWidth, imageHeight int, nnWidth, nnHeight int, minPadding int) Tiling {
	sx, nx := ComputeTileSpacingAndCount(imageWidth, nnWidth, minPadding)
	sy, ny := ComputeTileSpacingAndCount(imageHeight, nnHeight, minPadding)
	return Tiling{
		SpaceX: sx,
		NumX:   nx,
		SpaceY: sy,
		NumY:   ny,
	}
}

// Returns true if the tiling consists of just a single tile
func (t Tiling) IsSingle() bool {
	return t.NumX == 1 && t.NumY == 1
}

// Return the X,Y coordinates of the origin of the given tile
func (t Tiling) TileOrigin(x, y int) (int, int) {
	return OriginAt(x, t.SpaceX), OriginAt(y, t.SpaceY)
}

// Return the X1,Y1,X2,Y2 coordinates of the given tile
func (t Tiling) TileBox(x, y int) (int, int, int, int) {
	x1, y1 := t.TileOrigin(x, y)
	x2 := min(x1+t.NNWidth, t.ImageWidth)
	y2 := min(y1+t.NNHeight, t.ImageHeight)
	return x1, y1, x2, y2
}

// Return a single number that uniquely identifies this tile
func (t Tiling) MakeTileIndex(tx, ty int) int {
	return ty*t.NumX + tx
}

// Split a tile index created by MakeTileIndex into the original tx and ty that created it
func (t Tiling) SplitTileIndex(index int) (int, int) {
	return index % t.NumX, index / t.NumX
}

// Return the X or Y position of the origin of a tile
func OriginAt(i int, spaceBetween float64) int {
	return int(float64(i)*spaceBetween + 0.5)
}

// Split one dimension (either X or Y) into evenly spaced extents
// The returned float64 is the space between tiles.
// The returned int is the total number of tiles.
func ComputeTileSpacingAndCount(srcSize, nnSize, minPadding int) (float64, int) {
	if minPadding >= nnSize/2 {
		panic("Padding for tiled inference is too large")
	}
	if srcSize <= nnSize {
		return 0, 1
	}
	// This follows the discussion above, about exterior and interior tiles
	innerValid := srcSize - 2*(nnSize-minPadding)
	numInnerTiles := (innerValid + nnSize - minPadding*2 - 1) / (nnSize - minPadding*2) // round up
	numTotalTiles := 2 + numInnerTiles
	return float64(srcSize-nnSize) / float64(numTotalTiles-1), numTotalTiles
}
