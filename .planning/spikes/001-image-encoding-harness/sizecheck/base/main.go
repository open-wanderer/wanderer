package main

import (
	"image/jpeg"
	"os"

	"github.com/disintegration/imaging"
	
)

func main() {
	img, _ := imaging.Open(os.Args[1])
	_ = jpeg.Encode(os.Stdout, img, nil)
	
}
