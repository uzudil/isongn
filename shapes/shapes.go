package shapes

import (
	"fmt"
	"image"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"

	// import to initialize png decoding
	"image/draw"
	_ "image/png"
)

type Edge struct {
	Shapes []*Shape
}

type ShapeMeta struct {
	DpiMultiplier float32
	UnitPixels    [2]int
}

type TextureCoords struct {
	PixelOffset [2]float32
	PixelDim    [2]float32
	TexOffset   [2]float32
	TexDim      [2]float32
}

type Animation struct {
	Name  string
	Steps int
	Tex   map[Direction][]*TextureCoords
}

const alphaMinDefault = 0.35

type Shape struct {
	Index          int
	Name           string
	Group          int
	Image          *image.RGBA
	Size           [3]float32
	Tex            *TextureCoords
	Fudge          float32
	AlphaMin       float32
	ImageIndex     int
	ShapeMeta      *ShapeMeta
	Edges          map[string]map[string][]*Shape
	Offset         [3]float32
	EditorVisible  bool
	Animations     map[int]*Animation
	SwayEnabled    bool
	BobEnabled     bool
	BreatheEnabled bool
	NoSupport      bool
	IsExtra        bool
	IsDraggable    bool
	IsSaved        bool
}

var Shapes []*Shape
var Names map[string]int = map[string]int{}
var Images []image.Image
var UiImages map[string]image.Image = map[string]image.Image{}

// some pre-defined animations
const ANIMATION_MOVE = 0
const ANIMATION_STAND = 1
const ANIMATION_ATTACK = 2

// but more can be added
var AnimationNames map[string]int = map[string]int{
	"move":   ANIMATION_MOVE,
	"stand":  ANIMATION_STAND,
	"attack": ANIMATION_ATTACK,
}

func InitShapes(gameDir string, data []map[string]interface{}) error {
	for _, block := range data {
		imgFile := block["image"].(string)
		shapes := block["shapes"].([]interface{})
		fmt.Printf("Processing %s - %d shapes...\n", imgFile, len(shapes))

		// per-image meta data
		grid := block["grid"].(map[string]interface{})
		units := grid["units"].([]interface{})
		dpi := block["dpi"].(float64)
		shapeMeta := &ShapeMeta{
			DpiMultiplier: float32(dpi) / 96.0,
			UnitPixels:    [2]int{int(units[0].(float64)), int(units[1].(float64))},
		}

		img, err := loadImage(filepath.Join(gameDir, "images", imgFile))
		if err != nil {
			return err
		}
		imageIndex := len(Images)
		Images = append(Images, img)
		for index, s := range shapes {
			shapeDef := s.(map[string]interface{})
			name := shapeDef["name"].(string)
			appendShape(index, name, shapeDef, imageIndex, img, shapeMeta)
		}
		if imagesI, ok := block["images"]; ok {
			images := imagesI.([]interface{})
			for _, imageInfo := range images {
				imageDef := imageInfo.(map[string]interface{})
				appendUiImage(imageDef, img, shapeMeta)
			}
		}
	}
	fmt.Printf("Loaded %d shapes.\n", len(Shapes))
	return nil
}

func appendUiImage(imageDef map[string]interface{}, img image.Image, shapeMeta *ShapeMeta) {
	// size
	sizeI := imageDef["size"].([]interface{})
	size := [2]float32{float32(sizeI[0].(float64)), float32(sizeI[1].(float64))}

	// pixel bounding box
	posI := imageDef["pos"].([]interface{})
	px := float32(posI[0].(float64)) * shapeMeta.DpiMultiplier
	py := float32(posI[1].(float64)) * shapeMeta.DpiMultiplier
	pw := size[0] * shapeMeta.DpiMultiplier
	ph := size[1] * shapeMeta.DpiMultiplier

	// create a half-size thumbnail
	rgba := image.NewRGBA(image.Rect(0, 0, int(pw), int(ph)))
	draw.Draw(rgba, image.Rect(0, 0, int(pw), int(ph)), img, image.Point{int(px), int(py)}, draw.Src)
	w := int(float32(rgba.Bounds().Max.X) / shapeMeta.DpiMultiplier)
	h := int(float32(rgba.Bounds().Max.Y) / shapeMeta.DpiMultiplier)
	resized := imaging.Resize(rgba, w, h, imaging.NearestNeighbor)
	uiImage := image.NewRGBA(resized.Bounds())
	draw.Draw(uiImage, resized.Bounds(), resized, image.ZP, draw.Src)

	name := imageDef["name"].(string)
	UiImages[name] = uiImage
	fmt.Printf("\tStored UI Image: %s\n", name)
}

func appendShape(index int, name string, shapeDef map[string]interface{}, imageIndex int, img image.Image, shapeMeta *ShapeMeta) {
	// size
	sizeI := shapeDef["size"].([]interface{})
	size := [3]float32{float32(sizeI[0].(float64)), float32(sizeI[1].(float64)), float32(math.Max(sizeI[2].(float64), 0.1))}

	// pixel bounding box
	posI := shapeDef["pos"].([]interface{})
	px := float32(posI[0].(float64)) * shapeMeta.DpiMultiplier
	py := float32(posI[1].(float64)) * shapeMeta.DpiMultiplier
	unitPixelX := float32(shapeMeta.UnitPixels[0])
	unitPixelY := float32(shapeMeta.UnitPixels[1])
	pw := (size[0] + size[1]) * unitPixelX * shapeMeta.DpiMultiplier
	ph := (size[0] + size[1] + size[2]) * unitPixelY * shapeMeta.DpiMultiplier

	// fudge
	fudge64, ok := shapeDef["fudge"].(float64)
	var fudge float32 = 0
	if ok {
		fudge = float32(fudge64)
	}

	// alphaMin
	alphaMin64, ok := shapeDef["alphaMin"].(float64)
	var alphaMin float32 = alphaMinDefault
	if ok {
		alphaMin = float32(alphaMin64)
	}

	// offset
	offset := [3]float32{}
	if offsetI, ok := shapeDef["offset"].([]interface{}); ok {
		offset[0] = float32(offsetI[0].(float64))
		offset[1] = float32(offsetI[1].(float64))
		offset[2] = float32(offsetI[2].(float64))
	}

	groupF, ok := shapeDef["group"].(float64)
	var group int
	if ok {
		group = int(groupF)
	}

	shape := newShape(
		imageIndex*0x100+index, // each image can contain max 256 shapes
		name,
		group,
		size,
		px, py, pw, ph,
		img,
		fudge,
		alphaMin,
		imageIndex,
		shapeMeta,
		offset,
	)
	shape.addExtras(shapeDef)

	// edges
	refName, ok := shapeDef["ref"]
	if ok {
		targetI, ok := shapeDef["target"]
		var target string
		if ok == false {
			target = "default"
		} else {
			target = targetI.(string)
		}

		parts := strings.Split(name, ".")
		ref := findShape(refName.(string))
		if _, ok := ref.Edges[target]; ok == false {
			ref.Edges[target] = map[string][]*Shape{}
		}
		if _, ok := ref.Edges[target][parts[2]]; ok {
			ref.Edges[target][parts[2]] = append(ref.Edges[target][parts[2]], shape)
		} else {
			ref.Edges[target][parts[2]] = []*Shape{shape}
		}
	} else {
		shape.EditorVisible = true
	}

	// add a gap, if needed
	for len(Shapes) < shape.Index {
		Shapes = append(Shapes, nil)
	}
	// add shape
	Shapes = append(Shapes, shape)
	Names[name] = shape.Index
}

func (shape *Shape) addExtras(shapeDef map[string]interface{}) {
	// sway
	if sway, ok := shapeDef["sway"].(bool); ok {
		shape.SwayEnabled = sway
	}
	// bob
	if bob, ok := shapeDef["bob"].(bool); ok {
		shape.BobEnabled = bob
	}
	if bob, ok := shapeDef["breathe"].(bool); ok {
		shape.BreatheEnabled = bob
	}
	if bob, ok := shapeDef["nosupport"].(bool); ok {
		shape.NoSupport = bob
	}
	// extra
	if extra, ok := shapeDef["extra"].(bool); ok {
		shape.IsExtra = extra
	}
	// drag
	if drag, ok := shapeDef["drag"].(bool); ok {
		shape.IsDraggable = drag
	}
}

func (shape *Shape) HasEdges(shapeName string) bool {
	_, ok := shape.Edges[shapeName]
	if ok {
		return true
	}
	_, ok = shape.Edges["default"]
	if ok {
		return true
	}
	return false
}

func (shape *Shape) GetEdge(shapeName, edgeName string) *Shape {
	edgeMap, ok := shape.Edges[shapeName]
	if ok == false {
		edgeMap, ok = shape.Edges["default"]
	}
	if ok == false {
		fmt.Printf("No edges for shape %s\n", shape.Name)
		return nil
	}
	if edges, ok := edgeMap[edgeName]; ok {
		return edges[rand.Intn(len(edges))]
	}
	fmt.Printf("Can't find edge shape %s for %s\n", edgeName, shape.Name)
	return nil
}

func findShape(name string) *Shape {
	for _, s := range Shapes {
		if s.Name == name {
			return s
		}
	}
	panic("Can't find shape: " + name)
}

func newShape(index int, name string, group int, size [3]float32, px, py, pw, ph float32, img image.Image, fudge, alphaMin float32, imageIndex int, shapeMeta *ShapeMeta, offset [3]float32) *Shape {
	shape := &Shape{
		Index:      index,
		Name:       name,
		Group:      group,
		Size:       size,
		Tex:        NewTextureCoords(img.Bounds(), px, py, pw, ph),
		Fudge:      fudge,
		AlphaMin:   alphaMin,
		ImageIndex: imageIndex,
		ShapeMeta:  shapeMeta,
		Edges:      map[string]map[string][]*Shape{},
		Offset:     offset,
		IsSaved:    true,
	}

	// create a half-size thumbnail
	rgba := image.NewRGBA(image.Rect(0, 0, int(pw), int(ph)))
	draw.Draw(rgba, image.Rect(0, 0, int(pw), int(ph)), img, image.Point{int(px), int(py)}, draw.Src)
	w := int(float32(rgba.Bounds().Max.X) / shapeMeta.DpiMultiplier)
	h := int(float32(rgba.Bounds().Max.Y) / shapeMeta.DpiMultiplier)
	resized := imaging.Resize(rgba, w, h, imaging.NearestNeighbor)
	shape.Image = image.NewRGBA(resized.Bounds())
	draw.Draw(shape.Image, resized.Bounds(), resized, image.ZP, draw.Src)

	return shape
}

func NewTextureCoords(imageBounds image.Rectangle, px, py, pw, ph float32) *TextureCoords {
	return &TextureCoords{
		PixelOffset: [2]float32{px, py},
		PixelDim:    [2]float32{pw, ph},
		TexOffset:   [2]float32{float32(px) / float32(imageBounds.Max.X), float32(py) / float32(imageBounds.Max.Y)},
		TexDim:      [2]float32{float32(pw) / float32(imageBounds.Max.X), float32(ph) / float32(imageBounds.Max.Y)},
	}
}

func InitCreatures(gameDir string, data []map[string]interface{}) error {
	// create a large image to store all the animated textures
	for _, block := range data {
		name := block["name"].(string)
		fmt.Printf("\tProcessing creature: %s\n", name)
		img, err := loadImage(filepath.Join(gameDir, "creatures", fmt.Sprintf("%s.png", name)))
		if err != nil {
			return err
		}
		imageIndex := len(Images)
		Images = append(Images, img)

		sizeI := block["size"].([]interface{})
		size := [3]float32{float32(sizeI[0].(float64)), float32(sizeI[1].(float64)), float32(sizeI[2].(float64))}

		shape := &Shape{
			Index:      imageIndex * 0x100,
			Name:       name,
			Size:       size,
			ImageIndex: imageIndex,
			Animations: map[int]*Animation{},
			AlphaMin:   alphaMinDefault,
			IsSaved:    false,
		}
		shape.addExtras(block)

		// add a gap, if needed
		for len(Shapes) < shape.Index {
			Shapes = append(Shapes, nil)
		}
		Shapes = append(Shapes, shape)
		Names[name] = shape.Index

		dimI := block["dim"].([]interface{})
		dim := [2]int{int(dimI[0].(float64)), int(dimI[1].(float64))}

		frames := block["frames"].([]interface{})
		xpos := 0
		for _, frameBlock := range frames {
			frame := frameBlock.(map[string]interface{})
			a := &Animation{
				Name:  frame["name"].(string),
				Steps: int(frame["steps"].(float64)),
				Tex:   map[Direction][]*TextureCoords{},
			}
			dirs := frame["dirs"].([]interface{})
			for _, dirI := range dirs {
				dirFrames := []*TextureCoords{}
				for step := 0; step < a.Steps; step++ {
					dirFrames = append(dirFrames, NewTextureCoords(
						img.Bounds(),
						float32(xpos), 0,
						float32(dim[0]), float32(dim[1]),
					))
					if shape.Tex == nil {
						shape.Tex = dirFrames[0]
					}
					xpos += dim[0]
				}
				dir := dirI.(string)
				//fmt.Printf("\t\t\tadding %d steps for: %s\n", a.Steps, dir)
				a.Tex[Directions[dir]] = dirFrames
			}
			frameName := frame["name"].(string)
			animationIndex, ok := AnimationNames[frameName]
			if ok == false {
				animationIndex = len(AnimationNames)
				AnimationNames[frameName] = animationIndex
			}
			//fmt.Printf("\t\tadding animations for: %s\n", frameName)
			shape.Animations[animationIndex] = a
		}
	}
	return nil
}

func loadImage(path string) (image.Image, error) {
	imgFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer imgFile.Close()
	img, _, err := image.Decode(imgFile)
	return img, err
}

func (shape *Shape) Traverse(fx func(x, y, z int) bool) {
	for xx := 0; xx < int(shape.Size[0]); xx++ {
		for yy := 0; yy < int(shape.Size[1]); yy++ {
			for zz := 0; zz < int(shape.Size[2]); zz++ {
				if fx(xx, yy, zz) {
					return
				}
			}
		}
	}
}
