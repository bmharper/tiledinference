# TiledInference

TiledInference is a package for running a neural network Object Detector on an image
that is larger than your neural network. The package provides two mechanisms:

1. Slicing a large image up into evenly spaced tiles, with a guaranteed minimum overlap.
2. Merging object detections from the individual tiles back into a single set of detections.

Example:

```go
import "github.com/bmharper/tiledinference"

func main() {
	net := loadNeuralNetwork()
	img := loadImage()

	// Tiles will overlap by at least this number of pixels
	minOverlap := 32

	// Create a 'tiling', which is just 8 numbers that define how our image is divided
	ti := tiledinference.MakeTiling(net, img.Width, img.Height, nn.Width, nn.Height, minOverlap)

	// Run NN inference on each tile
	unmerged := []tiledinference.Box{}

	for ty := 0; ty < ti.NumY; ty++ {
		for tx := 0; tx < ti.NumX; tx++ {
			// extract a crop out of the image
			x1, y1, x2, y2 := ti.TileBox(tx, ty)
			crop := img.Crop(x1, y1, x2, y2)

			// run object detection
			boxes := net.DetectObjects(crop)
			for _, b := range boxes {
				// In your actual code you would be translating here between your internal
				// 'detection' format and the tiledinference.Box struct.
				// In this example, we pretend that nn.DetectObjects() returns tiledinference.Box structs.
				b.Tile = ti.MakeTileIndex(tx, ty)
				// Move the box coordinates into the original image reference frame
				b.Offset(x1, y1)
				unmerged = append(unmerged, b)
			}
		}
	}

	groups := tiledinference.MergeBoxes(unmerged, nil)

	// Here we could run some special merging logic, but in this simple
	// example we just take the first box in each group and discard the rest.
	output := []tiledinference.Box{}
	for _, g := range groups {
		first := unmerged[g[0]]
		output = append(output, first)
	}
}
```
