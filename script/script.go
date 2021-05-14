package script

import (
	"fmt"
	"math"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/uzudil/bscript/bscript"
	"github.com/uzudil/isongn/gfx"
	"github.com/uzudil/isongn/runner"
	"github.com/uzudil/isongn/shapes"
)

func getDateTime(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	runner := ctx.App["runner"].(*runner.Runner)
	return runner.Calendar.AsString(), nil
}

func getTime(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	runner := ctx.App["runner"].(*runner.Runner)
	return runner.Calendar.AsTimeString(), nil
}

func getCalendar(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	runner := ctx.App["runner"].(*runner.Runner)
	mins, hours, day, month, year := runner.Calendar.FromEpoch()
	r := make([]interface{}, 5)
	r[0] = float64(year)
	r[1] = float64(month)
	r[2] = float64(day)
	r[3] = float64(hours)
	r[4] = float64(mins)
	return &r, nil
}

func saveGame(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	app := ctx.App["app"].(*gfx.App)
	app.Loader.SaveAll()
	return nil, nil
}

func loadGame(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	return nil, nil
}

func intersectsShapes(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	ax := int(arg[0].(float64))
	ay := int(arg[1].(float64))
	az := int(arg[2].(float64))
	nameA := arg[3].(string)
	bx := int(arg[4].(float64))
	by := int(arg[5].(float64))
	bz := int(arg[6].(float64))
	nameB := arg[7].(string)
	shapeA := shapes.Shapes[shapes.Names[nameA]]
	shapeB := shapes.Shapes[shapes.Names[nameB]]
	if intersects(ax, ax+int(shapeA.Size[0]), bx, bx+int(shapeB.Size[0])) &&
		intersects(ay, ay+int(shapeA.Size[1]), by, by+int(shapeB.Size[1])) &&
		intersects(az, az+int(shapeA.Size[2]), bz, bz+int(shapeB.Size[2])) {
		return true, nil
	}
	return false, nil
}

func isInView(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	x := int(arg[0].(float64))
	y := int(arg[1].(float64))
	app := ctx.App["app"].(*gfx.App)
	return app.View.InView(x, y, 0), nil
}

func intersects(start, end, start2, end2 int) bool {
	return (end2 <= start || start2 >= end) == false
}

func setMaxZ(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	f := int(arg[0].(float64))
	app := ctx.App["app"].(*gfx.App)
	app.View.SetMaxZ(int(f))
	if roofName, ok := arg[1].(string); ok {
		roof := shapes.Shapes[shapes.Names[roofName]]
		app.View.SetUnderShape(roof)
	} else {
		app.View.SetUnderShape(nil)
	}
	return nil, nil
}

func print(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	fmt.Printf("%v\n", arg[0])
	return nil, nil
}

func eraseShape(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	x := int(arg[0].(float64))
	y := int(arg[1].(float64))
	z := int(arg[2].(float64))
	app := ctx.App["app"].(*gfx.App)
	app.View.EraseShape(x, y, z)
	return nil, nil
}

func setShape(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	x := int(arg[0].(float64))
	y := int(arg[1].(float64))
	z := int(arg[2].(float64))
	name := arg[3].(string)
	app := ctx.App["app"].(*gfx.App)
	app.View.SetShape(x, y, z, shapes.Names[name])
	return nil, nil
}

func setShapeExtra(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	x := int(arg[0].(float64))
	y := int(arg[1].(float64))
	z := int(arg[2].(float64))
	name := arg[3].(string)
	app := ctx.App["app"].(*gfx.App)
	app.Loader.AddExtra(x, y, z, shapes.Names[name])
	return nil, nil
}

func eraseAllExtras(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	x := int(arg[0].(float64))
	y := int(arg[1].(float64))
	z := int(arg[2].(float64))
	app := ctx.App["app"].(*gfx.App)
	app.Loader.EraseAllExtras(x, y, z)
	return nil, nil
}

func getShapeExtra(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	x := int(arg[0].(float64))
	y := int(arg[1].(float64))
	z := int(arg[2].(float64))
	app := ctx.App["app"].(*gfx.App)
	extras := app.Loader.GetExtras(x, y, z)
	r := make([]interface{}, len(extras))
	for i, e := range extras {
		r[i] = shapes.Shapes[e].Name
	}
	return &r, nil
}

func moveShape(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	x := int(arg[0].(float64))
	y := int(arg[1].(float64))
	z := int(arg[2].(float64))
	nx := int(arg[3].(float64))
	ny := int(arg[4].(float64))
	isFlying := arg[5].(bool)
	app := ctx.App["app"].(*gfx.App)
	return float64(app.View.MoveShape(x, y, z, nx, ny, isFlying)), nil
}

func setOffset(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	x := int(arg[0].(float64))
	y := int(arg[1].(float64))
	z := int(arg[2].(float64))
	sx := float32(arg[3].(float64))
	sy := float32(arg[4].(float64))
	app := ctx.App["app"].(*gfx.App)
	app.View.SetOffset(x, y, z, sx, sy)
	return nil, nil
}

func setViewScroll(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	sx := float32(arg[0].(float64))
	sy := float32(arg[1].(float64))
	sz := float32(arg[2].(float64))
	app := ctx.App["app"].(*gfx.App)
	app.View.Scroll(sx, sy, sz)
	return nil, nil
}

func setAnimation(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	x := int(arg[0].(float64))
	y := int(arg[1].(float64))
	z := int(arg[2].(float64))
	name := arg[3].(string)
	dir := arg[4].(float64)
	animationSpeed := arg[5].(float64)
	app := ctx.App["app"].(*gfx.App)
	app.View.SetShapeAnimation(x, y, z, shapes.AnimationNames[name], shapes.Direction(dir), animationSpeed)
	return nil, nil
}

func isEmpty(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	tx := int(arg[0].(float64))
	ty := int(arg[1].(float64))
	tz := int(arg[2].(float64))
	shape := arg[3].(string)
	app := ctx.App["app"].(*gfx.App)
	return app.View.IsEmpty(tx, ty, tz, shapes.Shapes[shapes.Names[shape]]), nil
}

func moveViewTo(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	x := int(arg[0].(float64))
	y := int(arg[1].(float64))
	app := ctx.App["app"].(*gfx.App)
	app.Loader.MoveTo(x, y)
	app.View.Load()
	return nil, nil
}

func fadeViewTo(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	x := int(arg[0].(float64))
	y := int(arg[1].(float64))
	app := ctx.App["app"].(*gfx.App)
	app.FadeOut(func() {
		app.FadeIn(func() {
			app.FadeDone()
		})
		app.Loader.MoveTo(x, y)
		app.View.Load()
	})
	return nil, nil
}

func getDir(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	dx := int(arg[0].(float64))
	dy := int(arg[1].(float64))
	return float64(shapes.GetDir(int(dx), int(dy))), nil
}

func getDelta(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	dir := shapes.Direction(int(arg[0].(float64)))
	dx, dy := dir.GetDelta()
	r := make([]interface{}, 2)
	r[0] = float64(dx)
	r[1] = float64(dy)
	return &r, nil
}

func getShape(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	x := int(arg[0].(float64))
	y := int(arg[1].(float64))
	z := int(arg[2].(float64))
	app := ctx.App["app"].(*gfx.App)
	if shapeIndex, ox, oy, oz, found := app.View.GetShape(x, y, z); found {
		r := make([]interface{}, 4)
		r[0] = shapes.Shapes[shapeIndex].Name
		r[1] = float64(ox)
		r[2] = float64(oy)
		r[3] = float64(oz)
		return &r, nil
	}
	return nil, nil
}

func getPosition(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	r := make([]interface{}, 3)
	app := ctx.App["app"].(*gfx.App)
	r[0] = float64(app.Loader.X)
	r[1] = float64(app.Loader.Y)
	r[2] = float64(app.Game.GetZ())
	return &r, nil
}

func isPressed(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	a, ok := arg[0].(float64)
	if ok {
		key := glfw.Key(int(a))
		app := ctx.App["app"].(*gfx.App)
		return app.IsFirstDown(key), nil
	}
	return nil, fmt.Errorf("%s unable to parse key at", ctx.Pos)
}

func showMessageAt(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	worldX := int(arg[0].(float64))
	worldY := int(arg[1].(float64))
	worldZ := int(arg[2].(float64))
	message := arg[3].(string)
	r := uint8(arg[4].(float64))
	g := uint8(arg[5].(float64))
	b := uint8(arg[6].(float64))
	runner := ctx.App["runner"].(*runner.Runner)
	runner.ShowMessageAt(worldX, worldY, worldZ, message, r, g, b)
	return nil, nil
}

func addMessage(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	x := int(arg[0].(float64))
	y := int(arg[1].(float64))
	message := arg[2].(string)
	r := uint8(arg[3].(float64))
	g := uint8(arg[4].(float64))
	b := uint8(arg[5].(float64))
	runner := ctx.App["runner"].(*runner.Runner)
	index := runner.AddMessage(x, y, message, r, g, b)
	return float64(index), nil
}

func delMessage(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	index := int(arg[0].(float64))
	runner := ctx.App["runner"].(*runner.Runner)
	runner.DelMessage(index)
	return nil, nil
}

func delAllMessages(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	runner := ctx.App["runner"].(*runner.Runner)
	runner.DelAllMessages()
	return nil, nil
}

func messageWidth(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	message := arg[0].(string)
	app := ctx.App["app"].(*gfx.App)
	return float64(app.Font.Width(message)), nil
}

func getScreenPos(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	x := int(arg[0].(float64))
	y := int(arg[1].(float64))
	z := int(arg[2].(float64))
	app := ctx.App["app"].(*gfx.App)
	sx, sy := app.GetScreenPos(x, y, z)
	r := make([]interface{}, 2)
	r[0] = float64(sx)
	r[1] = float64(sy)
	return &r, nil
}

func isDown(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	key := glfw.Key(arg[0].(float64))
	app := ctx.App["app"].(*gfx.App)
	return app.IsDown(key), nil
}

func distance(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	a := arg[0].(float64)
	b := arg[1].(float64)
	c := arg[2].(float64)
	x := arg[3].(float64)
	y := arg[4].(float64)
	z := arg[5].(float64)
	dx := (a - x) * (a - x)
	dy := (b - y) * (b - y)
	dz := (c - z) * (c - z)
	return math.Sqrt(dx + dy + dz), nil
}

func setPathThroughShapes(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	names := arg[0].(*[]interface{})
	app := ctx.App["app"].(*gfx.App)
	for _, name := range *names {
		app.View.AddPathThroughShape(shapes.Shapes[shapes.Names[name.(string)]])
	}
	return nil, nil
}

func findPath(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	sx := int(arg[0].(float64))
	sy := int(arg[1].(float64))
	sz := int(arg[2].(float64))
	ex := int(arg[3].(float64))
	ey := int(arg[4].(float64))
	ez := int(arg[5].(float64))
	isFlying := arg[6].(bool)
	app := ctx.App["app"].(*gfx.App)
	path := app.View.FindPath(sx, sy, sz, ex, ey, ez, isFlying)
	if path == nil {
		return nil, nil
	} else {
		r := make([]interface{}, len(path)*3)
		for i, node := range path {
			r[i*3] = float64(node[0])
			r[i*3+1] = float64(node[1])
			r[i*3+2] = float64(node[2])
		}
		return &r, nil
	}
}

func didClick(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	app := ctx.App["app"].(*gfx.App)
	return app.View.DidClick, nil
}

func getClick(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	app := ctx.App["app"].(*gfx.App)
	n := app.View.GetClick()
	r := make([]interface{}, 5)
	r[0] = float64(n[0])
	r[1] = float64(n[1])
	r[2] = float64(n[2])
	r[3] = app.Dragging
	r[4] = app.MouseButtonAction == 1
	app.CompleteDrag()
	return &r, nil
}

func findTop(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	worldX := int(arg[0].(float64))
	worldY := int(arg[1].(float64))
	shape := shapes.Shapes[shapes.Names[arg[2].(string)]]
	app := ctx.App["app"].(*gfx.App)
	z := app.View.FindTop(worldX, worldY, shape)
	if z < app.View.GetMaxZ() {
		return float64(z), nil
	} else {
		return float64(0), nil
	}
}

func setCursorShape(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	shapeIndex := shapes.Names[arg[0].(string)]
	app := ctx.App["app"].(*gfx.App)
	app.SetCursorShape(shapeIndex)
	return nil, nil
}

func clearCursorShape(ctx *bscript.Context, arg ...interface{}) (interface{}, error) {
	app := ctx.App["app"].(*gfx.App)
	app.HideCursorShape()
	return nil, nil
}

var constants map[string]interface{} = map[string]interface{}{
	// directions
	"DirW":    float64(shapes.DIR_W),
	"DirSW":   float64(shapes.DIR_SW),
	"DirS":    float64(shapes.DIR_S),
	"DirSE":   float64(shapes.DIR_SE),
	"DirE":    float64(shapes.DIR_E),
	"DirNE":   float64(shapes.DIR_NE),
	"DirN":    float64(shapes.DIR_N),
	"DirNW":   float64(shapes.DIR_NW),
	"DirNone": float64(shapes.DIR_NONE),
	// keyboard keys
	"KeyUnknown":      float64(glfw.KeyUnknown),
	"KeySpace":        float64(glfw.KeySpace),
	"KeyApostrophe":   float64(glfw.KeyApostrophe),
	"KeyComma":        float64(glfw.KeyComma),
	"KeyMinus":        float64(glfw.KeyMinus),
	"KeyPeriod":       float64(glfw.KeyPeriod),
	"KeySlash":        float64(glfw.KeySlash),
	"Key0":            float64(glfw.Key0),
	"Key1":            float64(glfw.Key1),
	"Key2":            float64(glfw.Key2),
	"Key3":            float64(glfw.Key3),
	"Key4":            float64(glfw.Key4),
	"Key5":            float64(glfw.Key5),
	"Key6":            float64(glfw.Key6),
	"Key7":            float64(glfw.Key7),
	"Key8":            float64(glfw.Key8),
	"Key9":            float64(glfw.Key9),
	"KeySemicolon":    float64(glfw.KeySemicolon),
	"KeyEqual":        float64(glfw.KeyEqual),
	"KeyA":            float64(glfw.KeyA),
	"KeyB":            float64(glfw.KeyB),
	"KeyC":            float64(glfw.KeyC),
	"KeyD":            float64(glfw.KeyD),
	"KeyE":            float64(glfw.KeyE),
	"KeyF":            float64(glfw.KeyF),
	"KeyG":            float64(glfw.KeyG),
	"KeyH":            float64(glfw.KeyH),
	"KeyI":            float64(glfw.KeyI),
	"KeyJ":            float64(glfw.KeyJ),
	"KeyK":            float64(glfw.KeyK),
	"KeyL":            float64(glfw.KeyL),
	"KeyM":            float64(glfw.KeyM),
	"KeyN":            float64(glfw.KeyN),
	"KeyO":            float64(glfw.KeyO),
	"KeyP":            float64(glfw.KeyP),
	"KeyQ":            float64(glfw.KeyQ),
	"KeyR":            float64(glfw.KeyR),
	"KeyS":            float64(glfw.KeyS),
	"KeyT":            float64(glfw.KeyT),
	"KeyU":            float64(glfw.KeyU),
	"KeyV":            float64(glfw.KeyV),
	"KeyW":            float64(glfw.KeyW),
	"KeyX":            float64(glfw.KeyX),
	"KeyY":            float64(glfw.KeyY),
	"KeyZ":            float64(glfw.KeyZ),
	"KeyLeftBracket":  float64(glfw.KeyLeftBracket),
	"KeyBackslash":    float64(glfw.KeyBackslash),
	"KeyRightBracket": float64(glfw.KeyRightBracket),
	"KeyGraveAccent":  float64(glfw.KeyGraveAccent),
	"KeyWorld1":       float64(glfw.KeyWorld1),
	"KeyWorld2":       float64(glfw.KeyWorld2),
	"KeyEscape":       float64(glfw.KeyEscape),
	"KeyEnter":        float64(glfw.KeyEnter),
	"KeyTab":          float64(glfw.KeyTab),
	"KeyBackspace":    float64(glfw.KeyBackspace),
	"KeyInsert":       float64(glfw.KeyInsert),
	"KeyDelete":       float64(glfw.KeyDelete),
	"KeyRight":        float64(glfw.KeyRight),
	"KeyLeft":         float64(glfw.KeyLeft),
	"KeyDown":         float64(glfw.KeyDown),
	"KeyUp":           float64(glfw.KeyUp),
	"KeyPageUp":       float64(glfw.KeyPageUp),
	"KeyPageDown":     float64(glfw.KeyPageDown),
	"KeyHome":         float64(glfw.KeyHome),
	"KeyEnd":          float64(glfw.KeyEnd),
	"KeyCapsLock":     float64(glfw.KeyCapsLock),
	"KeyScrollLock":   float64(glfw.KeyScrollLock),
	"KeyNumLock":      float64(glfw.KeyNumLock),
	"KeyPrintScreen":  float64(glfw.KeyPrintScreen),
	"KeyPause":        float64(glfw.KeyPause),
	"KeyF1":           float64(glfw.KeyF1),
	"KeyF2":           float64(glfw.KeyF2),
	"KeyF3":           float64(glfw.KeyF3),
	"KeyF4":           float64(glfw.KeyF4),
	"KeyF5":           float64(glfw.KeyF5),
	"KeyF6":           float64(glfw.KeyF6),
	"KeyF7":           float64(glfw.KeyF7),
	"KeyF8":           float64(glfw.KeyF8),
	"KeyF9":           float64(glfw.KeyF9),
	"KeyF10":          float64(glfw.KeyF10),
	"KeyF11":          float64(glfw.KeyF11),
	"KeyF12":          float64(glfw.KeyF12),
	"KeyF13":          float64(glfw.KeyF13),
	"KeyF14":          float64(glfw.KeyF14),
	"KeyF15":          float64(glfw.KeyF15),
	"KeyF16":          float64(glfw.KeyF16),
	"KeyF17":          float64(glfw.KeyF17),
	"KeyF18":          float64(glfw.KeyF18),
	"KeyF19":          float64(glfw.KeyF19),
	"KeyF20":          float64(glfw.KeyF20),
	"KeyF21":          float64(glfw.KeyF21),
	"KeyF22":          float64(glfw.KeyF22),
	"KeyF23":          float64(glfw.KeyF23),
	"KeyF24":          float64(glfw.KeyF24),
	"KeyF25":          float64(glfw.KeyF25),
	"KeyKP0":          float64(glfw.KeyKP0),
	"KeyKP1":          float64(glfw.KeyKP1),
	"KeyKP2":          float64(glfw.KeyKP2),
	"KeyKP3":          float64(glfw.KeyKP3),
	"KeyKP4":          float64(glfw.KeyKP4),
	"KeyKP5":          float64(glfw.KeyKP5),
	"KeyKP6":          float64(glfw.KeyKP6),
	"KeyKP7":          float64(glfw.KeyKP7),
	"KeyKP8":          float64(glfw.KeyKP8),
	"KeyKP9":          float64(glfw.KeyKP9),
	"KeyKPDecimal":    float64(glfw.KeyKPDecimal),
	"KeyKPDivide":     float64(glfw.KeyKPDivide),
	"KeyKPMultiply":   float64(glfw.KeyKPMultiply),
	"KeyKPSubtract":   float64(glfw.KeyKPSubtract),
	"KeyKPAdd":        float64(glfw.KeyKPAdd),
	"KeyKPEnter":      float64(glfw.KeyKPEnter),
	"KeyKPEqual":      float64(glfw.KeyKPEqual),
	"KeyLeftShift":    float64(glfw.KeyLeftShift),
	"KeyLeftControl":  float64(glfw.KeyLeftControl),
	"KeyLeftAlt":      float64(glfw.KeyLeftAlt),
	"KeyLeftSuper":    float64(glfw.KeyLeftSuper),
	"KeyRightShift":   float64(glfw.KeyRightShift),
	"KeyRightControl": float64(glfw.KeyRightControl),
	"KeyRightAlt":     float64(glfw.KeyRightAlt),
	"KeyRightSuper":   float64(glfw.KeyRightSuper),
	"KeyMenu":         float64(glfw.KeyMenu),
	"KeyLast":         float64(glfw.KeyLast),
}

func InitScript() {
	bscript.AddBuiltin("intersectsShapes", intersectsShapes)
	bscript.AddBuiltin("setMaxZ", setMaxZ)
	bscript.AddBuiltin("isPressed", isPressed)
	bscript.AddBuiltin("isDown", isDown)
	bscript.AddBuiltin("getPosition", getPosition)
	bscript.AddBuiltin("eraseShape", eraseShape)
	bscript.AddBuiltin("setShape", setShape)
	bscript.AddBuiltin("moveShape", moveShape)
	bscript.AddBuiltin("getShape", getShape)
	bscript.AddBuiltin("setShapeExtra", setShapeExtra)
	bscript.AddBuiltin("getShapeExtra", getShapeExtra)
	bscript.AddBuiltin("eraseAllExtras", eraseAllExtras)
	bscript.AddBuiltin("setAnimation", setAnimation)
	bscript.AddBuiltin("setOffset", setOffset)
	bscript.AddBuiltin("isEmpty", isEmpty)
	bscript.AddBuiltin("moveViewTo", moveViewTo)
	bscript.AddBuiltin("fadeViewTo", fadeViewTo)
	bscript.AddBuiltin("setViewScroll", setViewScroll)
	bscript.AddBuiltin("print", print)
	bscript.AddBuiltin("getDir", getDir)
	bscript.AddBuiltin("getDelta", getDelta)
	bscript.AddBuiltin("isInView", isInView)
	bscript.AddBuiltin("saveGame", saveGame)
	bscript.AddBuiltin("loadGame", loadGame)
	bscript.AddBuiltin("showMessageAt", showMessageAt)
	bscript.AddBuiltin("addMessage", addMessage)
	bscript.AddBuiltin("delMessage", delMessage)
	bscript.AddBuiltin("delAllMessages", delAllMessages)
	bscript.AddBuiltin("messageWidth", messageWidth)
	bscript.AddBuiltin("getDateTime", getDateTime)
	bscript.AddBuiltin("getTime", getTime)
	bscript.AddBuiltin("getCalendar", getCalendar)
	bscript.AddBuiltin("getScreenPos", getScreenPos)
	bscript.AddBuiltin("distance", distance)
	bscript.AddBuiltin("findPath", findPath)
	bscript.AddBuiltin("setPathThroughShapes", setPathThroughShapes)
	bscript.AddBuiltin("didClick", didClick)
	bscript.AddBuiltin("getClick", getClick)
	bscript.AddBuiltin("findTop", findTop)
	bscript.AddBuiltin("setCursorShape", setCursorShape)
	bscript.AddBuiltin("clearCursorShape", clearCursorShape)
	for k, v := range constants {
		bscript.AddConstant(k, v)
	}
}
