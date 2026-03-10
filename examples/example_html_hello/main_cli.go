//go:build !js

// Renders a simple HTML page to stdout.
package main

import "fmt"

func main() {
	fmt.Println(renderPage())
}
