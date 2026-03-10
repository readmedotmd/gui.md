//go:build js && wasm

package main

import "syscall/js"

func main() {
	html := renderPage()
	doc := js.Global().Get("document")
	doc.Call("open")
	doc.Call("write", html)
	doc.Call("close")
}
