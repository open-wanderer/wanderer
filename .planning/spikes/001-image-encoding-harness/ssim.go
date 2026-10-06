package main

import (
	"image"
	"image/color"
	"math"
)

// ssim returns the mean SSIM of the luma channel over 8x8 windows with stride 4.
// Good enough to compare encoders against each other on the same source; it is
// not a calibrated metric.
func ssim(ref, test image.Image) float64 {
	a, b := luma(ref), luma(test)
	w, h := ref.Bounds().Dx(), ref.Bounds().Dy()
	if test.Bounds().Dx() != w || test.Bounds().Dy() != h {
		return math.NaN()
	}
	const c1, c2 = 6.5025, 58.5225 // (0.01*255)^2, (0.03*255)^2
	var sum float64
	var n int
	for y := 0; y+8 <= h; y += 4 {
		for x := 0; x+8 <= w; x += 4 {
			var ma, mb float64
			for j := range 8 {
				for i := range 8 {
					ma += a[(y+j)*w+x+i]
					mb += b[(y+j)*w+x+i]
				}
			}
			ma /= 64
			mb /= 64
			var va, vb, cov float64
			for j := range 8 {
				for i := range 8 {
					da := a[(y+j)*w+x+i] - ma
					db := b[(y+j)*w+x+i] - mb
					va += da * da
					vb += db * db
					cov += da * db
				}
			}
			va /= 63
			vb /= 63
			cov /= 63
			sum += ((2*ma*mb + c1) * (2*cov + c2)) / ((ma*ma + mb*mb + c1) * (va + vb + c2))
			n++
		}
	}
	return sum / float64(n)
}

func luma(img image.Image) []float64 {
	b := img.Bounds()
	out := make([]float64, b.Dx()*b.Dy())
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			g := color.GrayModel.Convert(img.At(x, y)).(color.Gray)
			out[(y-b.Min.Y)*b.Dx()+(x-b.Min.X)] = float64(g.Y)
		}
	}
	return out
}
