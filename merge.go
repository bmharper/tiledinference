package tiledinference

import (
	"sort"

	flatbush "github.com/bmharper/flatbush-go"
)

// Rect is a rectangle
type Rect struct {
	X1 int32
	Y1 int32
	X2 int32
	Y2 int32
}

// Box is an object detection rectangle
type Box struct {
	Rect  Rect
	Tile  int   // Tile in which this box was detected
	Class int32 // Detection class
}

// Intersection over Union
func (r *Rect) IoU(b Rect) float64 {
	// Compute intersection
	x1 := max(r.X1, b.X1)
	y1 := max(r.Y1, b.Y1)
	x2 := min(r.X2, b.X2)
	y2 := min(r.Y2, b.Y2)
	intersection := float64(max(0, int(x2-x1))) * float64(max(0, int(y2-y1)))

	// Compute union
	area1 := float64(r.Area())
	area2 := float64(b.Area())
	union := area1 + area2 - intersection

	return intersection / union
}

func (r *Rect) Width() int {
	return int(r.X2 - r.X1)
}

func (r *Rect) Height() int {
	return int(r.Y2 - r.Y1)
}

func (r *Rect) Area() int {
	return r.Width() * r.Height()
}

// Move the tile by dx, dy
func (r *Rect) Offset(dx, dy int32) {
	r.X1 = r.X1 + dx
	r.Y1 = r.Y1 + dy
	r.X2 = r.X2 + dx
	r.Y2 = r.Y2 + dy
}

// Return a copy that has been clipped to clipper
func (r *Rect) ClipTo(clip Rect) Rect {
	return Rect{
		X1: max(r.X1, clip.X1),
		Y1: max(r.Y1, clip.Y1),
		X2: min(r.X2, clip.X2),
		Y2: min(r.Y2, clip.Y2),
	}
}

// Return the union of this box and b
func (r *Rect) Union(b Rect) Rect {
	return Rect{
		X1: min(r.X1, b.X1),
		Y1: min(r.Y1, b.Y1),
		X2: max(r.X2, b.X2),
		Y2: max(r.Y2, b.Y2),
	}
}

// MakeBox returns a Box
func MakeBox(x1, y1, x2, y2 int32, tile int, class int32) Box {
	return Box{
		Rect: Rect{
			X1: x1,
			Y1: y1,
			X2: x2,
			Y2: y2,
		},
		Tile:  tile,
		Class: class,
	}
}

// Object is the result of a neural network object detector
type Object interface {
	TiledInferenceBox() Box
}

// Options for merging objects
type MergeOptions struct {
	MinIoU                float64 // Minimum Intersection over Union for two boxes to be considered equal
	MergeDifferentClasses bool    // If true, boxes with different classes will be merged
}

// Return the default options that are used if the options parameter is nil
func DefaultMergeOptions() MergeOptions {
	return MergeOptions{
		MinIoU:                0.5,
		MergeDifferentClasses: false,
	}
}

// Merge objects
// options may be nil, in which case DefaultMergeOptions() is used.
// See MergeBoxes for a description of the return format.
func MergeObjects(tiling Tiling, objects []Object, options *MergeOptions) (groups [][]int, mergedBoxes []Box) {
	boxes := make([]Box, len(objects))
	for i, obj := range objects {
		boxes[i] = obj.TiledInferenceBox()
	}
	return MergeBoxes(tiling, boxes, options)
}

// Merge boxes
// options may be nil, in which case DefaultMergeOptions() is used.
// The returned data is a list of groups. Most groups will have just a single object
// inside it. Groups with more than one object inside mean those objects must be
// merged together.
// The integers inside the groups refers to the index of the object in the original query.
// The mergedBoxes array is equal in length to the groups array. The rectangles inside
// mergedBoxes will be the bounding box of the group. The class is the class of the
// first element in the group. The confidence is the maximum confidence of all the
// merged elements.
func MergeBoxes(tiling Tiling, boxes []Box, options *MergeOptions) (groups [][]int, mergedBoxes []Box) {
	defaultOptions := DefaultMergeOptions()
	if options == nil {
		options = &defaultOptions
	}

	// Create spatial index so that finding boxes in similar locations is fast
	fb := flatbush.NewFlatbush64()
	fb.Reserve(len(boxes))
	for _, b := range boxes {
		fb.Add(float64(b.Rect.X1), float64(b.Rect.Y1), float64(b.Rect.X2), float64(b.Rect.Y2))
	}
	fb.Finish()

	// Once a box is marked as 'consumed[i] = true', we don't touch it again.
	consumed := make([]bool, len(boxes))

	// flatMerge is one big array containing all groups. We use a single big array
	// so that we don't have a ton of small allocations. All of the groups that
	// we return are just slices into this array.
	flatMerge := []int{}

	// The starting index in flatMerge for each group.
	// Once we're finished, we add a terminator.
	groupStart := []int{}

	// nearby is recycled between iterations
	nearby := []int{}

	// We use this to avoid merging more than one object from every tile.
	// Such merging would imply that we're doing the job that the NMS was supposed
	// to play when the NN object detector originally ran.
	tilesInThisGroup := []int{}

	for i, r := range boxes {
		if consumed[i] {
			continue
		}
		consumed[i] = true
		nearby = fb.SearchFast(float64(r.Rect.X1), float64(r.Rect.Y1), float64(r.Rect.X2), float64(r.Rect.Y2), nearby)
		groupStart = append(groupStart, len(flatMerge))
		flatMerge = append(flatMerge, i)
		tilesInThisGroup = append(tilesInThisGroup[:0], int(r.Tile))

		// Clone the box, because when we merge it with other boxes, we're likely to expand it.
		// This applies when a large object is split across a tile boundary. Our padding helps
		// address somewhat, but it cannot get rid of it completely. Also, by being smart about
		// this, we get away with less padding, which improves performance.
		mergedBox := r

		// Sort nearby by tile index, and then by IoU.
		// This is so that when there are multiple candidate boxes in a neighbouring tile,
		// we pick the one that has the most overlap with the merged box.
		sort.Slice(nearby, func(a, b int) bool {
			if boxes[a].Tile != boxes[b].Tile {
				return boxes[a].Tile < boxes[b].Tile
			}
			aIoU := boxes[a].Rect.IoU(mergedBox.Rect)
			bIoU := boxes[b].Rect.IoU(mergedBox.Rect)
			return aIoU < bIoU
		})

		// The clipper gets smaller with every tile we merge into this group.
		clipper := tiling.TileRect(tiling.SplitTileIndex(r.Tile))

		for _, j := range nearby {
			if !consumed[j] &&
				(options.MergeDifferentClasses || boxes[i].Class == boxes[j].Class) &&
				indexOf(tilesInThisGroup, int(boxes[j].Tile)) == -1 {
				// We've passed the initial filter. Now we need to check that there is sufficient
				// overlap between the merged box and this box 'j'.
				// Before computing the IoU, we must clip the boxes to their NN extents.
				// Otherwise, if a sliver of a box is detected in a neighbouring tile, then it
				// will have a small IoU, and miss the opportunity to be merged, thus producing
				// duplicate detections with overlapping boxes.

				// Get the bounds of the tile that object 'j' came from
				jTileRect := tiling.TileRect(tiling.SplitTileIndex(boxes[j].Tile))
				newClipper := clipper.ClipTo(jTileRect)

				// Clip both boxes. You might think there's no need to clip the 'j' object box to its tile,
				// because surely the NN wouldn't output boxes outside of its bounds? This is not necessarily
				// true. For YOLOv8 for example, we can choose not to clip the boxes to the NN bounds.
				mergedClipped := mergedBox.Rect.ClipTo(newClipper)
				jClipped := boxes[j].Rect.ClipTo(newClipper)
				if mergedClipped.IoU(jClipped) >= options.MinIoU {
					// Merge j into this cluster
					consumed[j] = true
					mergedBox.Rect = mergedBox.Rect.Union(boxes[j].Rect)
					clipper = newClipper
					flatMerge = append(flatMerge, j)
					tilesInThisGroup = append(tilesInThisGroup, int(boxes[j].Tile))
				}
			}
		}
		mergedBoxes = append(mergedBoxes, mergedBox)
	}
	// Add terminator
	groupStart = append(groupStart, len(flatMerge))

	// Return results as a slice of slices. The inner slices all point inside flatMerge, so we're not doing
	// a ton of small allocations.
	groups = make([][]int, len(groupStart)-1)
	for i := 0; i < len(groups); i++ {
		groups[i] = flatMerge[groupStart[i]:groupStart[i+1]]
	}
	return groups, mergedBoxes
}

func indexOf(haystack []int, needle int) int {
	for i, v := range haystack {
		if v == needle {
			return i
		}
	}
	return -1
}
