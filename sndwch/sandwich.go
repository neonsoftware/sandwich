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
)

// Basic Types

type Cut2D struct {
	File string  // path to SVG file
	X    float64 // x offset, from left
	Y    float64 // y offset, from top
}

type Cut3D struct {
	Zmin float64 // where the cut should start, lowest point on Z axis, in mm
	Zmax float64 // where the cut should end, lowest point on Z axis, in mm
	Cut  Cut2D
}

type Layer struct {
	Zmin float64 // where the layer should start, lowest point on Z axis, in mm
	Zmax float64 // where the layer should start, lowest point on Z axis, in mm
	Cuts []Cut2D // All the cuts included in the layer
}

// simple toString functions

func (c Cut2D) String() string {
	return fmt.Sprintf("%v at (%v,%v)", c.File, c.X, c.Y)
}
func (c Cut3D) String() string {
	return fmt.Sprintf("[%v-%v] %v", c.Zmin, c.Zmax, c.Cut)
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

	var maxHeight float64 = 0.0
	var minHeight float64 = 0.0

	for _, cut3D := range cuts {
		if cut3D.Zmax > maxHeight {
			maxHeight = cut3D.Zmax
		}
		if cut3D.Zmin < minHeight {
			minHeight = cut3D.Zmin
		}
	}

	mmArranged2DCuts := make(map[float64][]Cut2D)

	for _, cut3D := range cuts {
		for mm := cut3D.Zmin; mm < cut3D.Zmax; mm += 0.5 {
			mmArranged2DCuts[mm] = append(mmArranged2DCuts[mm], cut3D.Cut)
		}
	}

	for mm := minHeight; mm < maxHeight; mm += 0.5 {
		layers = append(layers, Layer{mm, mm + 0.5, mmArranged2DCuts[mm]})
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
func WriteLayersToFile(outDirectory string, layers []Layer, x float64, y float64, xSizeMm float64, ySizeMm float64, groupParams string) (filesWritten []string) {

	filePaths := make([]string, 0)

	for _, l := range layers {
		fmt.Println("++++ This filtered layer is ", l)

		// Creating empty drawing
		fileName := fmt.Sprintf("design-%.1f-%.1f.svg", l.Zmin, l.Zmax)
		f, _ := os.Create(filepath.Join(outDirectory, fileName))
		defer f.Close()

		// TODO : check if float64 in w and h makes sense in SVG
		canvas := svg.New(f)
		canvas.StartviewUnit(float64(xSizeMm), float64(ySizeMm), "mm", x, y, float64(xSizeMm), float64(ySizeMm))
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

// WriteVisual, given a list of SVG files, creates an SVG file representinf the stack for the single
// input SVGs. A sort of sandwich. Hereby the library name.
func WriteVisual(outDirectory string, fileNames []string) {
	m, _ := os.Create(filepath.Join(outDirectory, "/design.svg"))
	defer m.Close()

	visual := svg.New(m)
	visual.Startraw("") // TODO : parametric size, of course
	visual.Group(`stroke="rgb(255,0,0)" stroke-width="1pt" fill="none"`)

	for i, f := range fileNames {
		fmt.Println("++++ File : ", f)
		y_insertion := len(fileNames) - i // we reverse, want the first one on the bottom, and so on ...
		importSvgElementsFromFile(visual, 50.0, 50+float64(y_insertion*100), f, "skewX(50)")
	}
	visual.Gend()
	visual.End()
}

// Makesandwich takes a slice of Cut3D struct 'cuts', creates a sandwich, and writes all the layers' SVG files
// to dir 'outDirectory'
func MakeSandwich(outDirectory string, all3DCuts []Cut3D, x float64, y float64, xSizeMm float64, ySizeMm float64, groupParams string) (string, error) {
	layersOnePerMM := SliceByMM(all3DCuts)
	finalLayers := MergeEqualLayers(layersOnePerMM)
	filePaths := WriteLayersToFile(outDirectory, finalLayers, x, y, xSizeMm, ySizeMm, groupParams)
	WriteVisual(outDirectory, filePaths)
	return "ok", nil
}
