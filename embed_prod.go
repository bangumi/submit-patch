//go:build !dev

package main

import "embed"

//go:embed view/templates/*
var templateFiles embed.FS
