package gfx

import (
	"fmt"
	"image"
	"image/draw"

	"github.com/go-gl/gl/all-core/gl"
	"github.com/uzudil/isongn/shapes"
)

type Texture struct {
	texture      uint32
	textureIndex int32
}

var textures map[int]*Texture = map[int]*Texture{}

func LoadTexture(imageIndex int) *Texture {
	tex, ok := textures[imageIndex]
	if ok == false {
		texID, err := loadTexture(shapes.Images[imageIndex])
		if err != nil {
			panic(err)
		}
		tex = &Texture{
			texture:      texID,
			textureIndex: int32(len(textures)),
		}
		textures[imageIndex] = tex
	}
	return tex
}

func loadTexture(img image.Image) (uint32, error) {
	// img := shapes.Images[0]
	rgba := image.NewRGBA(img.Bounds())
	if rgba.Stride != rgba.Rect.Size().X*4 {
		return 0, fmt.Errorf("unsupported stride")
	}
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)

	var texture uint32
	gl.GenTextures(1, &texture)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(rgba.Rect.Size().X),
		int32(rgba.Rect.Size().Y),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(rgba.Pix))

	return texture, nil
}
