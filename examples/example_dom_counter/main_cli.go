//go:build !js

// Prints a static HTML snapshot of the counter UI to stdout.
package main

import (
	"fmt"
	"github.com/readmedotmd/gui.md/html"
)

func main() {
	r := html.New()
	fmt.Println(r.RenderString(buildUI(0)))
}
