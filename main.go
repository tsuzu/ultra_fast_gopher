package main

import (
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"
	"io"
	"math"
	"os"

	"github.com/disintegration/imaging"
	"github.com/soniakeys/quant/median"
)

func main() {
	var ufpColor = []color.NRGBA{
		color.NRGBA{255, 130, 128, 255},
		color.NRGBA{125, 255, 126, 255},
		color.NRGBA{128, 172, 254, 255},
		color.NRGBA{255, 129, 255, 255},
		color.NRGBA{255, 94, 93, 255},
	}

	backgroundColor := color.RGBA{
		R: 0, G: 0, B: 0, A: 0,
	}

	generator, err := func() (func(fillColor, backgroundColor color.Color) image.Image, error) {
		fp, err := os.Open("gopherbw.png")
		//fp, err := os.Open("dman.png")

		if err != nil {
			return nil, err
		}
		defer fp.Close()

		gopherbw, err := png.Decode(fp)

		if err != nil {
			return nil, err
		}

		gopherbw = imaging.Resize(gopherbw, 200, 200, imaging.Linear)

		gophergs := gopherbw // imaging.Grayscale(gopherbw)

		mask := imaging.AdjustFunc(
			gophergs,
			func(c color.NRGBA) color.NRGBA {
				if c.A == 0 {
					return color.NRGBA{
						R: 255,
						G: 255,
						B: 255,
						A: 0,
					}
				}

				value := c.R // lightness
				if value < c.G {
					value = c.G
				}
				if value < c.B {
					value = c.B
				}

				return color.NRGBA{
					R: 255,
					G: 255,
					B: 255,
					A: 255 - value,
				}
			},
		)

		return func(fillColor, backgroundColor color.Color) image.Image {
			//dst := image.NewNRGBA(gophergs.Bounds())
			dst := imaging.New(gophergs.Bounds().Size().X, gophergs.Bounds().Size().Y, backgroundColor)

			draw.DrawMask(dst, dst.Bounds(), &image.Uniform{fillColor}, image.ZP, gophergs, image.ZP, draw.Over)
			draw.DrawMask(dst, dst.Bounds(), &image.Uniform{color.Black}, image.ZP, mask, image.ZP, draw.Over)

			return dst
		}, nil
	}()

	if err != nil {
		panic(err)
	}

	images := make([]image.Image, 0, len(ufpColor))
	var maxSize image.Point
	for i := range ufpColor {
		dst := generator(ufpColor[i], backgroundColor)
		images = append(images, dst)
	}

	gifGenerator := func(images []image.Image, writer io.Writer) error {
		if len(images) == 0 {
			return errors.New("no image")
		}

		agif := &gif.GIF{}
		for i := range images {

			quantizer := median.Quantizer(256)
			p := make(color.Palette, 0, 256)
			p = append(p, &color.NRGBA{0, 0, 0, 0})
			p = quantizer.Quantize(p, images[i])

			paletted := image.NewPaletted(images[i].Bounds(), p)

			draw.FloydSteinberg.Draw(paletted, paletted.Bounds(), images[i], image.ZP)

			agif.Image = append(agif.Image, paletted)
			agif.Delay = append(agif.Delay, 2)
			agif.Disposal = append(agif.Disposal, 2)
		}

		if len(agif.Image) == 0 {
			return nil
		}

		if err := gif.EncodeAll(writer, agif); err != nil {
			return err
		}

		return nil
	}

	aroundRotatingGenerator := func(path string, images []image.Image, backgroundColor color.Color) error {
		for i := range images {
			converted := image.NewRGBA(image.Rectangle{Max: images[i].Bounds().Size().Mul(3).Div(2)})

			const radius = 50
			movedRect := converted.Bounds().Sub(
				image.Point{
					X: int(-radius * math.Sin(2*math.Pi/float64(len(images))*float64(i))),
					Y: int(-radius * math.Cos(2*math.Pi/float64(len(images))*float64(i))),
				},
			).Add(converted.Bounds().Size().Sub(images[i].Bounds().Size()).Div(2))

			draw.Draw(converted, converted.Bounds(), &image.Uniform{backgroundColor}, image.ZP, draw.Src)
			draw.Draw(converted, movedRect, images[i], image.ZP, draw.Src)

			images[i] = converted
			dst := images[i]
			if maxSize.Y < dst.Bounds().Size().Y {
				maxSize.Y = dst.Bounds().Size().Y
			}
			if maxSize.X < dst.Bounds().Size().X {
				maxSize.X = dst.Bounds().Size().X
			}
		}

		for i := range images {
			resized := image.NewNRGBA(image.Rectangle{Max: maxSize})

			drawRect := resized.Bounds()
			drawRect.Min = drawRect.Max.Sub(images[i].Bounds().Size()).Div(2)
			drawRect.Max = drawRect.Max.Add(images[i].Bounds().Size()).Div(2)

			draw.Draw(resized, resized.Bounds(), &image.Uniform{backgroundColor}, image.ZP, draw.Src)
			draw.Draw(resized, drawRect, images[i], image.ZP, draw.Src)

			images[i] = resized
		}

		writer, err := os.Create(path)

		if err != nil {
			return err
		}
		defer writer.Close()

		if err := gifGenerator(images, writer); err != nil {
			return err
		}

		return nil
	}

	_ = /*rotatingGenerator := */ func(path string, images []image.Image, backgroundColor color.Color) error {
		for i := range images {
			images[i] = imaging.Rotate(images[i], 360./float64(len(images))*float64(i), backgroundColor)

			dst := images[i]
			if maxSize.Y < dst.Bounds().Size().Y {
				maxSize.Y = dst.Bounds().Size().Y
			}
			if maxSize.X < dst.Bounds().Size().X {
				maxSize.X = dst.Bounds().Size().X
			}
		}

		for i := range images {
			resized := image.NewNRGBA(image.Rectangle{Max: maxSize})

			drawRect := resized.Bounds()
			drawRect.Min = drawRect.Max.Sub(images[i].Bounds().Size()).Div(2)
			drawRect.Max = drawRect.Max.Add(images[i].Bounds().Size()).Div(2)

			draw.Draw(resized, resized.Bounds(), &image.Uniform{backgroundColor}, image.ZP, draw.Src)
			draw.Draw(resized, drawRect, images[i], image.ZP, draw.Src)

			images[i] = resized
		}

		writer, err := os.Create(path)

		if err != nil {
			return err
		}
		defer writer.Close()

		if err := gifGenerator(images, writer); err != nil {
			return err
		}

		return nil
	}

	if err := aroundRotatingGenerator("animeAround.gif", images, backgroundColor); err != nil {
		panic(err)
	}
}
