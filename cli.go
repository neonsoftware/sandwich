package main

import (
	"fmt"
	"github.com/docopt/docopt-go"
	"github.com/neonsoftware/sandwich/sndwch"
	"strconv"
	"strings"
)

// CLI interface
func main() {

	usage := `sandwich.

Usage:
  sandwich <out> <size_x> <size_y> <svg_path,x,y,z_min,z_out>...

Options:
  <out>                       The output directory
  <svg_path,x,y,z_min,z_out>  A 3DCut: a svg file and coordinates 
  <size_x>  				  Sandwich overall width, used as canvas size, in mm
  <size_y>                    Sandwich overall height, used as canvas size, in mm 
  -h --help                   Show this screen.
  -v --version                Show version.
  `

	arguments, _ := docopt.Parse(usage, nil, true, "0", false)
	fmt.Println("\n---> Arguments \n", arguments)

	var cuts []sandwich.Cut3D

	for _, svg_to_add := range arguments["<svg_path,x,y,z_min,z_out>"].([]string) {
		parts := strings.Split(svg_to_add, ",")
		if len(parts) != 5 {
			panic("2Dcut mal fomatted: " + svg_to_add)
		}

		x, err_x := strconv.ParseFloat(parts[1], 64)
		y, err_y := strconv.ParseFloat(parts[2], 64)
		zMin, err_zmin := strconv.ParseFloat(parts[3], 64)
		zMax, err_zmax := strconv.ParseFloat(parts[4], 64)

		if err_x != nil || err_y != nil || err_zmin != nil || err_zmax != nil {
			panic("2Dcut mal fomatted: " + svg_to_add)
		}

		path := parts[0]
		cuts = append(cuts, sandwich.Cut3D{zMin, zMax, sandwich.Cut2D{path, x, y}})

		sandwich.MakeSandwich(arguments["<out>"].(string), cuts, 0,0,arguments["<size_x>"].(float64), arguments["<size_y>"].(float64), `stroke="rgb(255,0,0)" stroke-width="0.2pt" fill="none"`)
	}

}
