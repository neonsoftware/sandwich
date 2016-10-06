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

// Basic Types

type Cut2D struct {
	File string  // path to SVG file
	X    float64 // x offset, from left
	Y    float64 // y offset, from top
}

func (c Cut2D) String() string {
	return fmt.Sprintf("%v at (%v,%v)", c.File, c.X, c.Y)
}

type Cut3D struct {
	Zmin int // where the cut should start, lowest point on Z axis, in mm
	Zmax int // where the cut should end, lowest point on Z axis, in mm
	Cut  Cut2D
}

func (c Cut3D) String() string {
	return fmt.Sprintf("[%v-%v] %v", c.Zmin, c.Zmax, c.Cut)
}

type Layer struct {
	Zmin int     // where the layer should start, lowest point on Z axis, in mm
	Zmax int     // where the layer should start, lowest point on Z axis, in mm
	Cuts []Cut2D // All the cuts included in the layer
}

func (l Layer) String() string {
	var inner string
	for _, c := range l.Cuts {
		inner += fmt.Sprintf("%v\n", c)
	}
	return fmt.Sprintf("z[%vmm-%vmm] : [%v]\n", l.Zmin, l.Zmax, inner)
}

// importSvgElementsFromFile reads an SVG and imports all the elements at (x,y) of currentDocument
func importSvgElementsFromFile(currentDocument *svg.SVG, x, y float64, fileName string, extra string) {

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

	currentDocument.Group(fmt.Sprintf(`transform="translate(%.2f,%.2f) %v"`, x, y, extra))
	io.WriteString(currentDocument.Writer, s.Doc)
	currentDocument.Gend()
}

func isCutEquivalent(a Cut2D, b Cut2D) bool {
	return a.File == b.File && a.X == b.X && a.Y == b.Y
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

	maxHeight := 0
	for _, cut3D := range cuts {
		if cut3D.Zmax > maxHeight {
			maxHeight = cut3D.Zmax
		}
	}

	mmArranged2DCuts := make([][]Cut2D, maxHeight+1)

	for _, cut3D := range cuts {
		for mm := cut3D.Zmin; mm <= cut3D.Zmax; mm++ {
			//_, thereIsAnotherAlready := mmArranged2DCuts[i]
			// if !thereIsAnotherAlready {
			// 	mmArranged2DCuts[i] = make([]Cut2D, 0)
			// }
			mmArranged2DCuts[mm] = append(mmArranged2DCuts[mm], cut3D.Cut)
		}
	}

	for mm, v := range mmArranged2DCuts {
		if mm > 0 {
			layers = append(layers, Layer{mm, mm, v})
		}
	}

	return layers
}

// MergeEqualLayers recognizes consecutive equivalent layers, which will be merged into a unique
// layer that combines the heights of the two.
func MergeEqualLayers(layers []Layer) []Layer {

	// TODO : take into account the non-consecutive equivalent. They will not have to be merged

	var filtered []Layer

	fmt.Println("To be filtered : ", layers)

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

// WriteLayersToFile, given a list of layers passed as input, writes an SVG file per layer.
// Such files will be written in the directory outDirectory passed as input.
// The format used for the SVG file names will contain the Zaxis, min and max, of the layer:
// <outDirectory>/design-A-B-mm.svg, where A is the min Z, and B is the max Z.
func WriteLayersToFile(outDirectory string, layers []Layer, xSizeMm float64, ySizeMm float64, groupParams string) (filesWritten []string) {

	filePaths := make([]string, 0)

	for _, l := range layers {
		fmt.Println("++++ This filtered layer is ", l)

		// Creating empty drawing
		f, _ := os.Create(filepath.Join(outDirectory, "/design-"+strconv.Itoa(l.Zmin)+"-"+strconv.Itoa(l.Zmax)+".svg"))
		defer f.Close()

		canvas := svg.New(f)
		canvas.StartviewUnit(xSizeMm, ySizeMm, "mm", 0, 0, xSizeMm, ySizeMm) // TODO : check if float64 in w and h makes sense in SVG
		canvas.Group(groupParams)
		for _, cut := range l.Cuts {
			importSvgElementsFromFile(canvas, cut.X, cut.Y, cut.File, "")
		}
		canvas.Gend()
		canvas.End()
		filePaths = append(filePaths, f.Name())
	}

	return filePaths
}

// WriteVisual, given a list of SVG files, creates an SVG file represneting the stack fo the single
// input SVGs. A sort of sandwich. Hereby the library name.
func WriteVisual(outDirectory string, layers []Layer, fileNames []string) {
	m, _ := os.Create(filepath.Join(outDirectory, "/design.svg"))
	defer m.Close()

	visual := svg.New(m)
	visual.Startraw("") // TODO : parametric size, of course
	visual.Group(`stroke="rgb(255,0,0)" stroke-width="1pt" fill="none"`)
	for i, f := range fileNames {
		fmt.Println("++++ File : ", f)
		importSvgElementsFromFile(visual, 50.0, 50+float64(i*100), f, "skewX(50)")
	}
	visual.Gend()
	visual.End()
}
