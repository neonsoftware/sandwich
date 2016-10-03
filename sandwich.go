// Licensed under terms of GPLv3 license
// Copyright (c) 2016 neonsoftware - neoncomputing eurl

package sandwich

import (
	"encoding/xml"
	"fmt"
	"github.com/ajstarks/svgo/float"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

type Cut2D struct {
	File string  // path to SVG file
	X    float64 // x offset, from left
	Y    float64 // y offset, from top
}

type Cut3D struct {
	Zmin int // where the cut should start, lowest point on Z axis, in mm
	Zmax int // where the cut should end, lowest point on Z axis, in mm
	Cut  Cut2D
}

type Layer struct {
	Zmin int     // where the layer should start, lowest point on Z axis, in mm
	Zmax int     // where the layer should start, lowest point on Z axis, in mm
	Cuts []Cut2D // All the cuts included in the layer
}

// Utility string dump functions (toString equivalents), used for debug.

func (c Cut2D) String() string {
	return fmt.Sprintf("%v at (%v,%v)", c.File, c.X, c.Y)
}

func (c Cut3D) String() string {
	return fmt.Sprintf("[%v-%v] %v", c.Zmin, c.Zmax, c.Cut)
}

func (l Layer) String() string {
	representation := fmt.Sprintf("[%v-%v] : ", l.Zmin, l.Zmax)
	for _, c := range l.Cuts {
		representation += fmt.Sprintf("%v\n", c)
	}
	return representation + "\n"
}

// importSvgElementsFromFile reads an SVG and imports all the elements at (x,y) of currentDocument
func importSvgElementsFromFile(currentDocument *svg.SVG, x, y float64, fileName string) {

	var s struct {
		Doc string `xml:",innerxml"`
	}

	f, err := os.Open(fileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}
	defer f.Close()
	if err := xml.NewDecoder(f).Decode(&s); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to parse (%v)\n", err)
		return
	}

	currentDocument.Group(fmt.Sprintf(`transform="translate(%.2f,%.2f)"`, x, y))
	io.WriteString(currentDocument.Writer, s.Doc)
	currentDocument.Gend()
}

// SliceByMM, given an array of Cut3D, creates an array of layers, one for every mm in the Z axis.
//
// The number of layers generated will therefore be the height, in mm, of the highest Cut3D object
// in the Z axis. If the highest Cut3D reaches 4mm, then 4 layers are generated (1mm,2mm,3mm,4mm).
// Iterating on every Cut3D, SliceByMM copies all the Cut2D objects included in the Cut3D object
// into every layer that the Cut3D trepasses.
//
// So if a Cut3D object with Zmin=2 and Zmax=4 is encountered, then its Cut2D objects are copied into
// the layer for Z 2mm, 3mm, and 4mm.
func SliceByMM(cuts []Cut3D) []Layer {

	var layers []Layer

	mmArranged2DCuts := make(map[int][]Cut2D)

	for _, cut3D := range cuts {
		for i := cut3D.Zmin; i <= cut3D.Zmax; i++ {
			//_, thereIsAnotherAlready := mmArranged2DCuts[i]
			// if !thereIsAnotherAlready {
			// 	mmArranged2DCuts[i] = make([]Cut2D, 0)
			// }
			mmArranged2DCuts[i] = append(mmArranged2DCuts[i], cut3D.Cut)
		}
	}

	for k, v := range mmArranged2DCuts {
		layers = append(layers, Layer{k, k, v})
	}

	return layers
}

func areCuts2DEquivalent(a []Cut2D, b []Cut2D) bool {
	if len(a) != len(b) {
		return false
	}

	for i, _ := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func MergeEqualLayers(layers []Layer) []Layer {

	var filtered []Layer

	for _, currentLayer := range layers {

		// Check if another is already in and can be merged
		merged := false
		for indexOfAlreadyPresentLayer, _ := range filtered {
			if areCuts2DEquivalent(currentLayer.Cuts, filtered[indexOfAlreadyPresentLayer].Cuts) {
				filtered[indexOfAlreadyPresentLayer].Zmax = currentLayer.Zmax
				merged = true
			}
		}

		if !merged {
			filtered = append(filtered, currentLayer)
		}
	}
	return filtered
}

func WriteLayersToFile(outDirectory string, layers []Layer) {
	for _, l := range layers {
		fmt.Println("++++ This filtered layer is [", l.Zmin, "-", l.Zmax, "]")

		// Creating empty drawing
		f, _ := os.Create(filepath.Join(outDirectory, "/design-"+strconv.Itoa(l.Zmin)+"-"+strconv.Itoa(l.Zmax)+".svg"))
		defer f.Close()

		canvas := svg.New(f)
		canvas.StartviewUnit(200.0, 200.0, "mm", 0, 0, 200, 200) // TODO : parametric size, of course
		canvas.Group(`stroke="rgb(255,0,0)" stroke-width="1pt" fill="none"`)

		for _, cut := range l.Cuts {
			importSvgElementsFromFile(canvas, cut.X, cut.Y, cut.File)
		}

		canvas.Gend()
		canvas.End()
	}
}
