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

// importSvgElementsFromFile reads an SVG and imports all the elements at (x,y) of currentDocument
func importSvgElementsFromFile(currentDocument *svg.SVG, x, y float64, filename string) {

	var s struct {
		Doc string `xml:",innerxml"`
	}

	f, err := os.Open(filename)
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

type Cut2D struct {
	File string  // from type's manifest
	X    float64 // x position within the device
	Y    float64 // x position within the device
}

type Cut3D struct {
	Zmin int   // from type's manifest
	Zmax int   // from type's manifest
	Cut  Cut2D // TODO : rename ?
}

type Layer struct {
	Zmin int     // from type's manifest
	Zmax int     // from type's manifest
	Cuts []Cut2D // TODO : rename ?
}

func Cut2DToString(c Cut2D) string {
	return fmt.Sprintf("%v at (%v,%v)", c.File, c.X, c.Y)
}

func Cut3DToString(c Cut3D) string {
	return fmt.Sprintf("[%v-%v] %v", c.Zmin, c.Zmax, Cut2DToString(c.Cut))
}

func LayerToString(l Layer) string {
	representation := fmt.Sprintf("[%v-%v] : ", l.Zmin, l.Zmax)
	for _, c := range l.Cuts {
		representation += Cut2DToString(c) + "\n"
	}
	return representation + "\n"
}

func SliceByMM(cuts3d []Cut3D) []Layer {

	var layers []Layer

	mmArranged2DCuts := make(map[int][]Cut2D)

	for _, cut3D := range cuts3d {
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

func MergeEqualLayers(inputLayers []Layer) []Layer {

	var filtered []Layer

	for _, currentLayer := range inputLayers {

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

func WriteLayersToFile(dir_path string, layers []Layer) {
	for _, l := range layers {
		fmt.Println("++++ This filtered layer is [", l.Zmin, "-", l.Zmax, "]")

		// Creating empty drawing
		f, _ := os.Create(filepath.Join(dir_path, "/design-"+strconv.Itoa(l.Zmin)+"-"+strconv.Itoa(l.Zmax)+".svg"))
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
