//go:build !dev

package main

import "embed"

//go:embed static/*
var staticFiles embed.FS

//go:embed view/templates/*
var templateFiles embed.FS
