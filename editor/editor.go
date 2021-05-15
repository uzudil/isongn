package editor

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"path/filepath"
	"strings"

	"github.com/uzudil/bscript/bscript"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/uzudil/isongn/gfx"
	"github.com/uzudil/isongn/shapes"
	"github.com/uzudil/isongn/util"
	"github.com/uzudil/isongn/world"
)

type Editor struct {
	app                 *gfx.App
	shapeSelectorIndex  int
	shapeSelectorUpdate bool
	infoUpdate          bool
	Z                   int
	ctx                 *bscript.Context
	editorCall          *bscript.Variable
	lastX, lastY        int
	updateCursor        bool
	startX, startY      int
}

func NewEditor(x, y int) *Editor {
	return &Editor{
		shapeSelectorUpdate: true,
		infoUpdate:          true,
		startX:              x,
		startY:              y,
	}
}

func (e *Editor) Init(app *gfx.App, config map[string]interface{}) {
	e.app = app
	e.app.FadeIn(func() {
		e.app.FadeDone()
	})
	e.app.Loader.SetIoMode(world.EDITOR_MODE)
	// e.app.Loader.MoveTo(4200, 4174)
	e.app.Loader.MoveTo(e.startX, e.startY)
	e.app.View.Load()

	// compile the editor script code
	ast, ctx, err := bscript.Build(
		filepath.Join(e.app.Config.GameDir, "src", "editor"),
		false,
		map[string]interface{}{
			"app":    app,
			"editor": e,
		},
	)
	if err != nil {
		panic(err)
	}
	e.ctx = ctx

	// create the editor bscript calls
	e.editorCall = util.NewFunctionCall("editorCommand")

	// run the main method
	_, err = ast.Evaluate(ctx)
	if err != nil {
		panic(err)
	}

	// add a ui
	e.app.Ui.Add(int(e.app.Width)-150, 0, 150, int(e.app.Height), e.shapeSelectorContents)
	e.app.Ui.Add(0, 0, int(e.app.Width)-150, 50, e.infoContents)
}

func (e *Editor) Name() string {
	return "editor"
}

var moveKeys map[glfw.Key][2]int = map[glfw.Key][2]int{
	glfw.KeyA:     {1, 0},
	glfw.KeyLeft:  {1, 0},
	glfw.KeyD:     {-1, 0},
	glfw.KeyRight: {-1, 0},
	glfw.KeyW:     {0, -1},
	glfw.KeyUp:    {0, -1},
	glfw.KeyS:     {0, 1},
	glfw.KeyDown:  {0, 1},
}

func (e *Editor) isMoveKey() (int, int, bool) {
	shape := shapes.Shapes[e.shapeSelectorIndex]
	for key, delta := range moveKeys {
		f := e.app.IsFirstDown(key)
		if f && e.app.IsDownMod(key, glfw.ModAlt) {
			return delta[0] * int(shape.Size[0]), delta[1] * int(shape.Size[1]), true
		}
		if f && e.app.IsDownMod(key, glfw.ModSuper) {
			return delta[0] * int(shape.Size[0]), delta[1] * int(shape.Size[1]), false
		}
		if f || e.app.IsDownMod(key, glfw.ModShift) {
			return delta[0], delta[1], false
		}
	}
	return 0, 0, false
}

func (e *Editor) Events(delta float64, fadeDir int, mouseX, mouseY int32) {

	if e.app.Loader.X != e.lastX || e.app.Loader.Y != e.lastY || e.updateCursor {
		// e.Z = e.findTop(e.app.Loader.X, e.app.Loader.Y)
		e.app.View.SetCursor(e.shapeSelectorIndex, e.Z)
		e.lastX = e.app.Loader.X
		e.lastY = e.app.Loader.Y
		e.updateCursor = false
	}

	if e.app.IsFirstDown(glfw.KeyPeriod) && e.Z < world.SECTION_Z_SIZE-1 {
		e.Z++
		e.updateCursor = true
	}
	if e.app.IsFirstDown(glfw.KeyComma) && e.Z > 0 {
		e.Z--
		e.updateCursor = true
	}

	if e.app.IsDownAlt1(glfw.KeyLeftBracket) && e.shapeSelectorIndex > 0 {
		for i := e.shapeSelectorIndex - 1; i >= 0; i-- {
			if shapes.Shapes[i] != nil && shapes.Shapes[i].EditorVisible {
				e.shapeSelectorIndex = i
				e.shapeSelectorUpdate = true
				break
			}
		}
	}
	if e.app.IsDownAlt1(glfw.KeyRightBracket) && e.shapeSelectorIndex < len(shapes.Shapes)-1 {
		for i := e.shapeSelectorIndex + 1; i < len(shapes.Shapes); i++ {
			if shapes.Shapes[i] != nil && shapes.Shapes[i].EditorVisible {
				e.shapeSelectorIndex = i
				e.shapeSelectorUpdate = true
				break
			}
		}
	}

	// shape := shapes.Shapes[e.shapeSelectorIndex]
	changed := false
	dx, dy, insertMode := e.isMoveKey()
	// if insertMode {
	// 	dx *= int(shape.Size[0])
	// 	dy *= int(shape.Size[1])
	// }

	f := e.app.IsFirstDown(glfw.KeySpace)
	ff := e.app.IsDownMod(glfw.KeySpace, glfw.ModShift)
	if insertMode || f || ff {
		e.setShape(e.app.Loader.X, e.app.Loader.Y, e.Z, shapes.Shapes[e.shapeSelectorIndex], ff)
		changed = true
	}
	if e.app.IsFirstDownMod(glfw.KeyE, glfw.ModShift) {
		shapes.Shapes[e.shapeSelectorIndex].Traverse(func(xx, yy, zz int) bool {
			e.app.Loader.EraseAllExtras(e.app.Loader.X+xx, e.app.Loader.Y+yy, e.Z+zz)
			return false
		})
		changed = true
	} else if e.app.IsFirstDown(glfw.KeyE) && e.Z > 0 {
		shapes.Shapes[e.shapeSelectorIndex].Traverse(func(xx, yy, zz int) bool {
			if zz == 0 {
				e.app.View.EraseShape(e.app.Loader.X+xx, e.app.Loader.Y+yy, e.Z-1)
				changed = true
			}
			return false
		})
	}
	if e.app.IsFirstDown(glfw.KeyU) {
		seen := map[string]bool{}
		shapes.Shapes[e.shapeSelectorIndex].Traverse(func(xx, yy, zz int) bool {
			if zz == 0 {
				wx, wy, wz := e.app.Loader.X+xx, e.app.Loader.Y+yy, e.Z+zz-1
				if shapeIndex, ox, oy, oz, ok := e.app.View.GetShape(wx, wy, wz); ok {
					k := fmt.Sprintf("%d.%d.%d", ox, oy, oz)
					if _, ok := seen[k]; !ok {
						fmt.Printf("%s at %d,%d,%d\n", shapes.Shapes[shapeIndex].Name, ox, oy, oz)
						seen[k] = true
					}
				}
			}
			return false
		})
		changed = true
	}

	if e.app.IsFirstDown(glfw.KeyX) {
		e.app.Loader.SaveAll()
	}
	if e.app.IsFirstDown(glfw.KeyF) {
		e.fill()
	}

	// call bscript
	e.editorCall.Evaluate(e.ctx)

	// move
	if e.app.Loader.MoveTo(e.app.Loader.X+dx, e.app.Loader.Y+dy) {
		e.app.View.Load()
		e.Z = e.findTop(e.app.Loader.X, e.app.Loader.Y)
		e.infoUpdate = true
	}

	if changed || e.shapeSelectorUpdate {
		e.updateCursor = true
	}
}

func (e *Editor) GetZ() int {
	return e.Z
}

func (e *Editor) fill() {
	shape := shapes.Shapes[e.shapeSelectorIndex]
	if strings.HasPrefix(shape.Name, "ground.") {
		shapeIndex, _, _, _, found := e.app.View.GetShape(e.app.Loader.X, e.app.Loader.Y, 0)
		var replaceShape *shapes.Shape
		if found {
			replaceShape = shapes.Shapes[shapeIndex]
		}
		fmt.Printf("replaceShape=%v\n", replaceShape)
		e.fillAt(e.app.Loader.X, e.app.Loader.Y, shape, replaceShape, map[string]bool{})
	}
}

func (e *Editor) fillAt(x, y int, shape, replaceShape *shapes.Shape, seen map[string]bool) {
	if e.app.View.InView(x, y, 0) == false {
		return
	}
	key := fmt.Sprintf("%d.%d", x, y)
	if _, ok := seen[key]; ok {
		return
	}
	seen[key] = true
	shapeIndex, _, _, _, found := e.app.View.GetShape(x, y, 0)
	fmt.Printf("\tpos=%d,%d found=%v\n", x, y, found)
	if (replaceShape == nil && found == false) || (replaceShape != nil && int(shapeIndex) == replaceShape.Index) {
		e.setShape(x, y, 0, shape, false)
		w := int(shape.Size[0])
		h := int(shape.Size[1])
		e.fillAt(x-w, y, shape, replaceShape, seen)
		e.fillAt(x+w, y, shape, replaceShape, seen)
		e.fillAt(x, y-h, shape, replaceShape, seen)
		e.fillAt(x, y+h, shape, replaceShape, seen)
	}
}

func (e *Editor) setShape(x, y, z int, shape *shapes.Shape, skipEdge bool) {
	if strings.HasPrefix(shape.Name, "ground.") {
		w := int(shape.Size[0])
		h := int(shape.Size[1])
		x = (x / w) * w
		y = (y / h) * h
		z = 0
	}
	if shape.IsExtra {
		e.app.Loader.AddExtra(x, y, z, e.shapeSelectorIndex)
	} else {
		e.app.View.SetShape(x, y, z, e.shapeSelectorIndex)
	}

	if skipEdge {
		e.app.Loader.ClearEdge(x, y)
	} else {
		if z == 0 {
			for xx := -1; xx <= 1; xx++ {
				for yy := -1; yy <= 1; yy++ {
					sx := x + xx*int(shape.Size[0])
					sy := y + yy*int(shape.Size[1])
					shapeIndex, ox, oy, _, found := e.app.View.GetShape(sx, sy, z)
					if found {
						e.setEdges(ox, oy, shapes.Shapes[shapeIndex])
					}
				}
			}
		}
	}
}

func (e *Editor) setEdges(x, y int, shape *shapes.Shape) {
	e.app.Loader.ClearEdge(x, y)

	w := int(shape.Size[0])
	h := int(shape.Size[1])

	shapeN := e.getEdgeShape(x, y-h, shape)
	shapeS := e.getEdgeShape(x, y+h, shape)
	shapeE := e.getEdgeShape(x-w, y, shape)
	shapeW := e.getEdgeShape(x+w, y, shape)

	var edgeShape *shapes.Shape
	var edgeName string = ""
	if shapeN != nil && shapeS != nil && shapeE != nil && shapeW != nil {
		edgeShape = shapeN
		edgeName = "nsew"
	} else if shapeN != nil && shapeS != nil && shapeE != nil {
		edgeShape = shapeN
		edgeName = "nse"
	} else if shapeN != nil && shapeS != nil && shapeW != nil {
		edgeShape = shapeN
		edgeName = "nsw"
	} else if shapeE != nil && shapeS != nil && shapeW != nil {
		edgeShape = shapeS
		edgeName = "sew"
	} else if shapeE != nil && shapeN != nil && shapeW != nil {
		edgeShape = shapeN
		edgeName = "new"
	} else if shapeE != nil && shapeW != nil {
		edgeShape = shapeE
		edgeName = "ew"
	} else if shapeN != nil && shapeS != nil {
		edgeShape = shapeN
		edgeName = "ns"
	} else if shapeN != nil && shapeE != nil {
		edgeShape = shapeN
		edgeName = "ne"
	} else if shapeN != nil && shapeW != nil {
		edgeShape = shapeN
		edgeName = "nw"
	} else if shapeS != nil && shapeE != nil {
		edgeShape = shapeS
		edgeName = "se"
	} else if shapeS != nil && shapeW != nil {
		edgeShape = shapeS
		edgeName = "sw"
	} else if shapeN != nil {
		edgeShape = shapeN
		edgeName = "n"
	} else if shapeS != nil {
		edgeShape = shapeS
		edgeName = "s"
	} else if shapeE != nil {
		edgeShape = shapeE
		edgeName = "e"
	} else if shapeW != nil {
		edgeShape = shapeW
		edgeName = "w"
	}

	if edgeName != "" && edgeShape.Index != shape.Index {
		edge := edgeShape.GetEdge(shape.Name, edgeName)
		if edge != nil {
			e.app.Loader.SetEdge(x, y, edge.Index)
		}
	}
}

func (e *Editor) getEdgeShape(x, y int, target *shapes.Shape) *shapes.Shape {
	shapeIndex, _, _, _, found := e.app.View.GetShape(x, y, 0)
	if found == false {
		return nil
	}
	shape := shapes.Shapes[shapeIndex]
	if shape.HasEdges(target.Name) {
		return shape
	}
	return nil
}

func (e *Editor) findTop(worldX, worldY int) int {
	return e.app.View.FindTop(worldX, worldY, shapes.Shapes[e.shapeSelectorIndex])
}

func (e *Editor) infoContents(panel *gfx.Panel) bool {
	if e.infoUpdate {
		panel.Clear()
		sx, sy := e.app.Loader.GetSectionPos()
		e.app.Font.Printf(panel.Rgba, color.Black, 0, 30, "pos=%d,%d,%d section=%d,%d", e.app.Loader.X, e.app.Loader.Y, e.Z, sx, sy)
		e.infoUpdate = false
		return true
	}
	return false
}

func (e *Editor) shapeSelectorContents(panel *gfx.Panel) bool {
	if e.shapeSelectorUpdate {
		panel.Clear()
		y := 0
		for i := e.shapeSelectorIndex; i < len(shapes.Shapes) && y < panel.H; i++ {
			shape := shapes.Shapes[i]
			if shape != nil && shape.EditorVisible {
				shapeW := shape.Image.Bounds().Dx()
				shapeH := shape.Image.Bounds().Dy()
				if i == e.shapeSelectorIndex {
					draw.Draw(panel.Rgba, image.Rect(0, y, 150, y+shapeH), &image.Uniform{color.RGBA{0xff, 0xa0, 0, 0xff}}, image.ZP, draw.Src)
				}
				draw.Draw(panel.Rgba, image.Rect(0, y, shapeW, y+shapeH), shape.Image, image.Point{0, 0}, draw.Over)
				y += shape.Image.Bounds().Dy()
			}
		}
		e.shapeSelectorUpdate = false
		return true
	}
	return false
}

func (e *Editor) SectionLoad(x, y int, data map[string]interface{}) {
}

func (e *Editor) SectionSave(x, y int) map[string]interface{} {
	return map[string]interface{}{}
}

func (e *Editor) DragFromUi(pixelX, pixelY int) (string, int) {
	return "", 0
}
