package gfx

// Credit: copied from https://github.com/nullboundary/glfont project. Thank you!

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io/ioutil"
	"log"
	"os"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

const (
	lowChar  = 32
	highChar = 127
)

type character struct {
	width    int //glyph width
	height   int //glyph height
	advance  int //glyph advance
	bearingH int //glyph bearing horizontal
	bearingV int //glyph bearing vertical
	rgba     *image.RGBA
}

type Font struct {
	chars  []*character
	Height int
}

func NewFont(fontPath string, scale int) (*Font, error) {
	fmt.Printf("Reading font file: %s\n", fontPath)
	file, err := os.Open(fontPath)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err = file.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	// Read the truetype font.
	ttf, err := truetype.Parse(data)
	if err != nil {
		return nil, err
	}

	f := &Font{
		Height: scale,
	}
	f.chars = make([]*character, 0, highChar-lowChar+1)
	for ch := lowChar; ch <= highChar; ch++ {
		char, err := newChar(ttf, ch, scale)
		if err != nil {
			return nil, err
		}
		f.chars = append(f.chars, char)
	}
	return f, nil
}

func newChar(ttf *truetype.Font, ch, scale int) (*character, error) {
	char := new(character)

	//create new face to measure glyph diamensions
	ttfFace := truetype.NewFace(ttf, &truetype.Options{
		Size:    float64(scale),
		DPI:     72,
		Hinting: font.HintingFull,
	})

	gBnd, gAdv, ok := ttfFace.GlyphBounds(rune(ch))
	if ok != true {
		return nil, fmt.Errorf("ttf face glyphBounds error")
	}

	gh := int32((gBnd.Max.Y - gBnd.Min.Y) >> 6)
	gw := int32((gBnd.Max.X - gBnd.Min.X) >> 6)

	//if gylph has no diamensions set to a max value
	if gw == 0 || gh == 0 {
		gBnd = ttf.Bounds(fixed.Int26_6(scale))
		gw = int32((gBnd.Max.X - gBnd.Min.X) >> 6)
		gh = int32((gBnd.Max.Y - gBnd.Min.Y) >> 6)

		//above can sometimes yield 0 for font smaller than 48pt, 1 is minimum
		if gw == 0 || gh == 0 {
			gw = 1
			gh = 1
		}
	}

	//The glyph's ascent and descent equal -bounds.Min.Y and +bounds.Max.Y.
	gAscent := int(-gBnd.Min.Y) >> 6
	gdescent := int(gBnd.Max.Y) >> 6

	//set w,h and adv, bearing V and bearing H in char
	char.width = int(gw)
	char.height = int(gh)
	char.advance = int(gAdv)
	char.bearingV = gdescent
	char.bearingH = (int(gBnd.Min.X) >> 6)

	//create image to draw glyph
	fg, bg := image.White, image.Transparent
	rect := image.Rect(0, 0, int(gw), int(gh))
	char.rgba = image.NewRGBA(rect)
	draw.Draw(char.rgba, char.rgba.Bounds(), bg, image.ZP, draw.Src)

	//create a freetype context for drawing
	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(ttf)
	c.SetFontSize(float64(scale))
	c.SetClip(char.rgba.Bounds())
	c.SetDst(char.rgba)
	c.SetSrc(fg)
	c.SetHinting(font.HintingFull)

	//set the glyph dot
	px := 0 - (int(gBnd.Min.X) >> 6)
	py := (gAscent)
	pt := freetype.Pt(px, py)

	// Draw the text from mask to image
	_, err := c.DrawString(string(ch), pt)
	if err != nil {
		return nil, err
	}

	return char, nil
}

// Width returns the width of a piece of text in pixels
func (f *Font) Width(fs string, argv ...interface{}) float32 {

	var width float32

	indices := []rune(fmt.Sprintf(fs, argv...))

	if len(indices) == 0 {
		return 0
	}

	// Iterate through all characters in string
	for i := range indices {

		//get rune
		runeIndex := indices[i]

		//skip runes that are not in font chacter range
		if int(runeIndex)-lowChar > len(f.chars) || runeIndex < lowChar {
			fmt.Printf("CHAR NOT FOUND: %c %d\n", runeIndex, runeIndex)
			continue
		}

		//find rune in fontChar list
		ch := f.chars[runeIndex-lowChar]

		// Now advance cursors for next glyph (note that advance is number of 1/64 pixels)
		width += float32((ch.advance >> 6)) // Bitshift by 6 to get value in pixels (2^6 = 64 (divide amount of 1/64th pixels by 64 to get amount of pixels))

	}

	return width
}

func (f *Font) Printf(dst draw.Image, fg color.Color, x, y int, fs string, argv ...interface{}) {
	indices := []rune(fmt.Sprintf(fs, argv...))

	if len(indices) == 0 {
		return
	}

	src := image.NewUniform(fg)
	for i := range indices {

		//get rune
		runeIndex := indices[i]

		//skip runes that are not in font chacter range
		if int(runeIndex)-int(lowChar) > len(f.chars) || runeIndex < lowChar {
			fmt.Printf("CHAR NOT FOUND %c %d\n", runeIndex, runeIndex)
			continue
		}

		//find rune in fontChar list
		ch := f.chars[runeIndex-lowChar]

		xpos := x + ch.bearingH
		// ypos := y + ch.height - ch.bearingV
		ypos := y - ch.height + ch.bearingV

		draw.DrawMask(dst, image.Rect(xpos, ypos, xpos+ch.width, ypos+ch.height), src, image.ZP, ch.rgba, image.ZP, draw.Over)

		// Now advance cursors for next glyph (note that advance is number of 1/64 pixels)
		x += (ch.advance >> 6) // Bitshift by 6 to get value in pixels (2^6 = 64 (divide amount of 1/64th pixels by 64 to get amount of pixels))
	}
}
