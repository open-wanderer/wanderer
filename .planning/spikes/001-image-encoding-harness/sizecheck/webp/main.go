package main

import (
	"image/jpeg"
	"os"

	"github.com/disintegration/imaging"
	"github.com/gen2brain/webp"
)

func main() {
	img, _ := imaging.Open(os.Args[1])
	_ = jpeg.Encode(os.Stdout, img, nil)
	_ = webp.Encode
}
