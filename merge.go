package tiledinference

import flatbush "github.com/bmharper/flatbush-go"

// Rect is an object detection rectangle
type Rect struct {
	X1    int32
	Y1    int32
	X2    int32
	Y2    int32
	Class int32 // Detection class
}

// Intersection over Union
func (r Rect) IoU(b Rect) float64 {
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

func (r Rect) Width() float64 {
	return float64(r.X2 - r.X1)
}

func (r Rect) Height() float64 {
	return float64(r.Y2 - r.Y1)
}

func (r Rect) Area() float64 {
	return float64(r.Width()) * float64(r.Height())
}

// Object is something that can be represented as a rectangle
type Object interface {
	Rect() Rect
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
	boxes := make([]Rect, len(objects))
	for i, obj := range objects {
		boxes[i] = obj.Rect()
	}
	return MergeBoxes(boxes, options)
}

// Merge boxes
// options may be nil, in which case DefaultMergeOptions() is used.
// The returned data is a list of groups. Most groups will have just a single object
// inside it. Groups with more than one object inside mean those objects must be
// merged together.
// The integers inside the groups refers to the index of the object in the original query.
func MergeBoxes(boxes []Rect, options *MergeOptions) [][]int {
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
	for i, r := range boxes {
		if consumed[i] {
			continue
		}
		nearby = fb.SearchFast(float64(r.X1), float64(r.Y1), float64(r.X2), float64(r.Y2), nearby)
		groupStart = append(groupStart, len(flatMerge))
		flatMerge = append(flatMerge, i)
		for _, j := range nearby {
			if i != j && !consumed[j] && (options.MergeDifferentClasses || boxes[i].Class == boxes[j].Class) {
				if r.IoU(boxes[j]) >= options.MinIoU {
					flatMerge = append(flatMerge, j)
					consumed[j] = true
				}
			}
		}
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
