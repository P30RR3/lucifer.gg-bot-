package config

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"sync"
)

const (
	infernoWidth  = 960
	infernoHeight = 320
)

var (
	infernoImageOnce sync.Once
	infernoImagePNG  []byte
	infernoImageErr  error
)

func infernoBackgroundPNG() ([]byte, error) {
	infernoImageOnce.Do(func() {
		infernoImagePNG, infernoImageErr = renderInfernoBackground()
	})
	return infernoImagePNG, infernoImageErr
}

func renderInfernoBackground() ([]byte, error) {
	canvas := image.NewRGBA(image.Rect(0, 0, infernoWidth, infernoHeight))
	for y := 0; y < infernoHeight; y++ {
		t := float64(y) / float64(infernoHeight-1)
		for x := 0; x < infernoWidth; x++ {
			heat := math.Max(0, 1-math.Abs(float64(x-infernoWidth/2)/float64(infernoWidth/2)))
			glow := heat * math.Pow(t, 3) * 92
			canvas.SetRGBA(x, y, color.RGBA{
				R: uint8(clamp(13+74*t+glow, 0, 255)),
				G: uint8(clamp(12+13*t+glow*0.17, 0, 255)),
				B: uint8(clamp(31-12*t+glow*0.04, 0, 255)),
				A: 255,
			})
		}
	}

	for x := 0; x < infernoWidth; x++ {
		horizon := 250 + int(13*math.Sin(float64(x)*0.018)) + int(8*math.Sin(float64(x)*0.043+1.3))
		for y := horizon; y < infernoHeight; y++ {
			heat := uint8(clamp(114+float64(infernoHeight-y)*1.3, 0, 255))
			canvas.SetRGBA(x, y, color.RGBA{R: heat, G: heat / 5, B: 18, A: 255})
		}
	}

	for x := 0; x < infernoWidth; x++ {
		ridge := 276 + int(19*math.Sin(float64(x)*0.009+0.4)) + int(12*math.Sin(float64(x)*0.031))
		for y := ridge; y < infernoHeight; y++ {
			canvas.SetRGBA(x, y, color.RGBA{R: 31, G: 12, B: 27, A: 255})
		}
	}

	for index := 0; index < 110; index++ {
		x := (index*137 + 41) % infernoWidth
		y := 65 + (index*71+17)%210
		size := 1 + index%3
		ember := color.RGBA{R: 255, G: uint8(90 + index%110), B: 37, A: 255}
		for dy := -size; dy <= size; dy++ {
			for dx := -size; dx <= size; dx++ {
				if dx*dx+dy*dy <= size*size {
					canvas.SetRGBA(x+dx, y+dy, ember)
				}
			}
		}
	}

	var output bytes.Buffer
	if err := png.Encode(&output, canvas); err != nil {
		return nil, fmt.Errorf("encode inferno background as PNG: %w", err)
	}
	if output.Len() == 0 {
		return nil, errors.New("encoded inferno background is empty")
	}
	return output.Bytes(), nil
}

func clamp(value, min, max float64) float64 {
	return math.Min(math.Max(value, min), max)
}
