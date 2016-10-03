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
	file string  // from type's manifest
	x    float64 // x position within the device
	y    float64 // x position within the device
}

type Cut3D struct {
	zMin int   // from type's manifest
	zMax int   // from type's manifest
	cut  Cut2D // TODO : rename ?
}

type Layer struct {
	zMin int     // from type's manifest
	zMax int     // from type's manifest
	cuts []Cut2D // TODO : rename ?
}

func Cut2DToString(c Cut2D) string {
	return fmt.Sprintf("%v at (%v,%v)", c.file, c.x, c.y)
}

func Cut3DToString(c Cut3D) string {
	return fmt.Sprintf("[%v-%v] %v", c.zMin, c.zMax, Cut2DToString(c.cut))
}

func LayerToString(l Layer) string {
	representation := fmt.Sprintf("[%v-%v] : ", l.zMin, l.zMax)
	for _, c := range l.cuts {
		representation += Cut2DToString(c) + "\n"
	}
	return representation + "\n"
}

func sliceByMM(cuts3d []Cut3D) []Layer {

	var layers []Layer

	mmArranged2DCuts := make(map[int][]Cut2D)

	for _, cut3D := range cuts3d {
		for i := cut3D.zMin; i <= cut3D.zMax; i++ {
			//_, thereIsAnotherAlready := mmArranged2DCuts[i]
			// if !thereIsAnotherAlready {
			// 	mmArranged2DCuts[i] = make([]Cut2D, 0)
			// }
			mmArranged2DCuts[i] = append(mmArranged2DCuts[i], cut3D.cut)
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

func mergeEqualLayers(inputLayers []Layer) []Layer {

	var filtered []Layer

	for _, currentLayer := range inputLayers {

		// Check if another is already in and can be merged
		merged := false
		for indexOfAlreadyPresentLayer, _ := range filtered {
			if areCuts2DEquivalent(currentLayer.cuts, filtered[indexOfAlreadyPresentLayer].cuts) {
				filtered[indexOfAlreadyPresentLayer].zMax = currentLayer.zMax
				merged = true
			}
		}

		if !merged {
			filtered = append(filtered, currentLayer)
		}
	}
	return filtered
}

func writeLayersToFile(dir_path string, layers []Layer) {
	for _, l := range layers {
		fmt.Println("++++ This filtered layer is [", l.zMin, "-", l.zMax, "]")

		// Creating empty drawing
		f, _ := os.Create(filepath.Join(dir_path, "/design-"+strconv.Itoa(l.zMin)+"-"+strconv.Itoa(l.zMax)+".svg"))
		defer f.Close()

		canvas := svg.New(f)
		canvas.StartviewUnit(200.0, 200.0, "mm", 0, 0, 200, 200) // TODO : parametric size, of course
		canvas.Group(`stroke="rgb(255,0,0)" stroke-width="1pt" fill="none"`)

		for _, cut := range l.cuts {
			importSvgElementsFromFile(canvas, cut.x, cut.y, cut.file)
		}

		canvas.Gend()
		canvas.End()
	}
}
