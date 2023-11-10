package tiledinference

import flatbush "github.com/bmharper/flatbush-go"

// Box is an object detection rectangle
type Box struct {
	X1    int32
	Y1    int32
	X2    int32
	Y2    int32
	Tile  int   // Tile in which this box was detected
	Class int32 // Detection class
}

// MakeBox returns a Box
func MakeBox(x1, y1, x2, y2 int32, tile int, class int32) Box {
	return Box{
		X1:    x1,
		Y1:    y1,
		X2:    x2,
		Y2:    y2,
		Tile:  tile,
		Class: class,
	}
}

// Intersection over Union
func (r Box) IoU(b Box) float64 {
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

func (r Box) Width() float64 {
	return float64(r.X2 - r.X1)
}

func (r Box) Height() float64 {
	return float64(r.Y2 - r.Y1)
}

func (r Box) Area() float64 {
	return float64(r.Width()) * float64(r.Height())
}

// Move the tile by dx, dy
func (r *Box) Offset(dx, dy int32) {
	r.X1 = r.X1 + dx
	r.Y1 = r.Y1 + dy
	r.X2 = r.X2 + dx
	r.Y2 = r.Y2 + dy
}

// Object is something that can be represented as a rectangle
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
func MergeObjects(objects []Object, options *MergeOptions) [][]int {
	boxes := make([]Box, len(objects))
	for i, obj := range objects {
		boxes[i] = obj.TiledInferenceBox()
	}
	return MergeBoxes(boxes, options)
}

// Merge boxes
// options may be nil, in which case DefaultMergeOptions() is used.
// The returned data is a list of groups. Most groups will have just a single object
// inside it. Groups with more than one object inside mean those objects must be
// merged together.
// The integers inside the groups refers to the index of the object in the original query.
func MergeBoxes(boxes []Box, options *MergeOptions) [][]int {
	defaultOptions := DefaultMergeOptions()
	if options == nil {
		options = &defaultOptions
	}

	// Create spatial index and 'consumed' array
	fb := flatbush.NewFlatbush64()
	fb.Reserve(len(boxes))
	for _, r := range boxes {
		fb.Add(float64(r.X1), float64(r.Y1), float64(r.X2), float64(r.Y2))
	}
	fb.Finish()

	// Merge boxes
	consumed := make([]bool, len(boxes))
	flatMerge := []int{}
	groupStart := []int{}
	nearby := []int{}
	tilesInThisGroup := []int{}
	for i, r := range boxes {
		if consumed[i] {
			continue
		}
		nearby = fb.SearchFast(float64(r.X1), float64(r.Y1), float64(r.X2), float64(r.Y2), nearby)
		groupStart = append(groupStart, len(flatMerge))
		flatMerge = append(flatMerge, i)
		tilesInThisGroup = tilesInThisGroup[:0]
		for _, j := range nearby {
			if i != j &&
				!consumed[j] &&
				boxes[i].Tile != boxes[j].Tile &&
				(options.MergeDifferentClasses || boxes[i].Class == boxes[j].Class) &&
				r.IoU(boxes[j]) >= options.MinIoU &&
				indexOf(tilesInThisGroup, int(boxes[j].Tile)) == -1 {
				// Merge j into this cluster
				flatMerge = append(flatMerge, j)
				consumed[j] = true
				tilesInThisGroup = append(tilesInThisGroup, int(boxes[j].Tile))
			}
		}
		consumed[i] = true
	}
	// Add terminator
	groupStart = append(groupStart, len(flatMerge))

	// Return results as a slice of slices. The inner slices all point inside flatMerge, so we're not doing
	// a ton of small allocations.
	merge := make([][]int, len(groupStart)-1)
	for i := 0; i < len(merge); i++ {
		merge[i] = flatMerge[groupStart[i]:groupStart[i+1]]
	}
	return merge
}

func indexOf(haystack []int, needle int) int {
	for i, v := range haystack {
		if v == needle {
			return i
		}
	}
	return -1
}
