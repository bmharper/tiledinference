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

	// Create a TiledInference object
	minOverlap := 32
	ti := tiledinference.MakeTiling(net, img.Width, img.Height, nn.Width, nn.Height, minOverlap)

	unmerged := []tiledinference.Box{}

	for ty := 0; ty < ti.NumY; ty++ {
		for tx := 0; tx < ti.NumX; tx++ {
			txo, tyo := ti.TilePosition(tx, ty)
			crop := img.Crop(txo, tyo, txo + nn.Width, tyo + nn.Width)
			boxes := net.DetectObjects(crop)
			for _, b := range boxes {
				b.Tile = int32(ty * ti.NumX + tx)
				unmerged = append(unmerged, b)
			}
		}
	}

	groups := tiledinference.MergeBoxes(unmerged, nil)

	// Here we could run some special merging logic, but in this simple
	// example we just take the first box in each group and discard the rest.
	final := []tiledinference.Box{}
	for _, g := range groups {
		final = append(final, unmerged[g[0]])
	}
}
```
