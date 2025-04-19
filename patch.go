package main

import (
	"github.com/aymanbagabas/go-udiff"
)

func Diff(name, before, after string) string {
	return udiff.Unified(name, name, before, after)
}

const PatchTypeSubject string = "subject"
const PatchTypeEpisode string = "episode"
