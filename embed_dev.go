//go:build dev

package main

import "os"

var staticFiles = os.DirFS(".")
var templateFiles = os.DirFS(".")
