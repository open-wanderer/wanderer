package main

import (
	"image/jpeg"
	"os"

	"github.com/disintegration/imaging"
	"github.com/gen2brain/jpegli"
)

func main() {
	img, _ := imaging.Open(os.Args[1])
	_ = jpeg.Encode(os.Stdout, img, nil)
	_ = jpegli.Encode
}
