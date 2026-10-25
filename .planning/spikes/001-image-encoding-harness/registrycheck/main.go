// Which decoder does the global image registry use once jpegli is linked in?
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"time"

	"github.com/disintegration/imaging"
	_ "github.com/gen2brain/jpegli"
)

func main() {
	data, _ := os.ReadFile("fixtures/24mp.jpg")

	t := time.Now()
	_, err := jpeg.Decode(bytes.NewReader(data))
	fmt.Println("stdlib jpeg.Decode:", time.Since(t).Round(time.Millisecond), err)

	t = time.Now()
	img, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	fmt.Printf("imaging.Decode (PocketBase thumbs): %v err=%v type=%T\n", time.Since(t).Round(time.Millisecond), err, img)

	func() {
		defer func() { fmt.Println("image.DecodeConfig recovered panic:", recover() != nil) }()
		_, format, err := image.DecodeConfig(bytes.NewReader(data))
		fmt.Println("image.DecodeConfig:", format, err)
	}()

	// A truncated upload through the full decoder.
	func() {
		defer func() { fmt.Println("truncated imaging.Decode recovered panic:", recover() != nil) }()
		_, err := imaging.Decode(bytes.NewReader(data[:len(data)/2]))
		fmt.Println("truncated imaging.Decode err:", err)
	}()
}
