package gfx

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-gl/gl/all-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/uzudil/isongn/shapes"
	"github.com/uzudil/isongn/world"
)

const fadeInterval = 0.5
const minDragPixels = 5
const dontReadPos = 0
const readDragPos = 1
const readMousePos = 2

type Game interface {
	Init(app *App, config map[string]interface{})
	Name() string
	Events(delta float64, fadeDir int, mouseX, mouseY int32)
	GetZ() int
	DragFromUi(pixelX, pixelY int) (string, int)
}

type KeyPress struct {
	Key      glfw.Key
	Scancode int
	Action   glfw.Action
	Mods     glfw.ModifierKey
	First    bool
}

type AppConfig struct {
	GameDir    string
	Title      string
	Name       string
	Version    float64
	ViewSize   int
	ViewSizeZ  int
	SectorSize int
	runtime    map[string]interface{}
	zoom       float64
	camera     [3]float32
	shear      [3]float32
	shapes     []map[string]interface{}
	creatures  []map[string]interface{}
}

type App struct {
	Game                                 Game
	Fonts                                []*Font
	Config                               *AppConfig
	Window                               *glfw.Window
	KeyState                             map[glfw.Key]*KeyPress
	targetFps                            float64
	lastUpdate                           float64
	nbFrames                             int
	View                                 *View
	Ui                                   *Ui
	Dir                                  string
	Loader                               *world.Loader
	Width, Height                        int
	windowWidth, windowHeight            int
	windowWidthDpi, windowHeightDpi      int
	dpiX, dpiY                           float32
	pxWidth, pxHeight                    int
	frameBuffer, uiFrameBuffer           *FrameBuffer
	fadeDir                              int
	fadeTimer                            float64
	fadeFx                               func()
	fade                                 float32
	MouseX, MouseY                       int32
	MousePixelX, MousePixelY             int32
	DragStartX, DragStartY               int32
	readSelection                        int
	MouseButtonAction                    int
	Dragging                             bool
	DraggedPanel                         *Panel
	DraggedPanelOffsX, DraggedPanelOffsY int
	DragAction                           string
	DragIndex                            int
	cursorPanel                          *Panel
	Loading                              bool
}

func NewApp(game Game, gameDir string, windowWidth, windowHeight int, targetFps float64) *App {
	// make sure the ./game/maps dir exists
	mapDir := filepath.Join(gameDir, "maps")
	if _, err := os.Stat(mapDir); os.IsNotExist(err) {
		os.Mkdir(mapDir, os.ModePerm)
	}
	appConfig := parseConfig(gameDir)
	width, height := getResolution(appConfig, game.Name())
	app := &App{
		Game:         game,
		Config:       appConfig,
		KeyState:     map[glfw.Key]*KeyPress{},
		targetFps:    targetFps,
		Width:        width,
		Height:       height,
		windowWidth:  windowWidth,
		windowHeight: windowHeight,
		Fonts:        []*Font{},
	}
	app.addFonts(appConfig, gameDir, game.Name())
	app.Dir = initUserdir(appConfig.Name)
	app.Window = initWindow(windowWidth, windowHeight)
	app.pxWidth, app.pxHeight = app.Window.GetFramebufferSize()
	app.dpiX = float32(app.pxWidth) / float32(windowWidth)
	app.dpiY = float32(app.pxHeight) / float32(windowHeight)
	fmt.Printf("Resolution: %dx%d Window: %dx%d Dpi: %fx%f\n", app.Width, app.Height, windowWidth, windowHeight, app.dpiX, app.dpiY)
	app.windowWidthDpi = int(float32(app.windowWidth) * app.dpiX)
	app.windowHeightDpi = int(float32(app.windowHeight) * app.dpiY)
	app.Window.SetKeyCallback(app.Keypressed)
	app.Window.SetScrollCallback(app.MouseScroll)
	app.Window.SetCursorPosCallback(app.MousePos)
	app.Window.SetMouseButtonCallback(app.MouseClick)
	app.frameBuffer = NewFrameBuffer(int32(width), int32(height), true)
	app.uiFrameBuffer = NewFrameBuffer(int32(width), int32(height), false)
	err := shapes.InitShapes(gameDir, appConfig.shapes)
	if err != nil {
		panic(err)
	}
	err = shapes.InitCreatures(gameDir, appConfig.creatures)
	if err != nil {
		panic(err)
	}
	app.Loader = world.NewLoader(game.(world.WorldObserver), app.Dir, gameDir)
	app.View = InitView(appConfig.zoom, appConfig.camera, appConfig.shear, app.Loader)
	app.Ui = InitUi(width, height)
	return app
}

func (app *App) GetScreenPos(x, y, z int) (int, int) {
	if sx, sy, ok := app.View.toScreenPos(x, y, z, app.Width, app.Height); ok {
		return sx, sy
	}
	// offscreen
	return -1000, -1000
}

func parseConfig(gameDir string) *AppConfig {
	configPath := filepath.Join(gameDir, "config.json")
	bytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		panic(err)
	}
	data := map[string]interface{}{}
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		panic(err)
	}

	view := data["view"].(map[string]interface{})
	camera := view["camera"].([]interface{})
	shear := view["shear"].([]interface{})
	config := &AppConfig{
		GameDir:    gameDir,
		Title:      data["title"].(string),
		Name:       strings.ToLower(data["name"].(string)),
		Version:    data["version"].(float64),
		ViewSize:   int(view["size"].(float64)),
		ViewSizeZ:  int(view["sizeZ"].(float64)),
		SectorSize: int(view["sector"].(float64)),
		runtime:    data["runtime"].(map[string]interface{}),
		zoom:       view["zoom"].(float64),
		camera:     [3]float32{float32(camera[0].(float64)), float32(camera[1].(float64)), float32(camera[2].(float64))},
		shear:      [3]float32{float32(shear[0].(float64)), float32(shear[1].(float64)), float32(shear[2].(float64))},
		shapes:     toMap(data["shapes"].([]interface{})),
		creatures:  toMap(data["creatures"].([]interface{})),
	}
	fmt.Printf("Starting game: %s (v%f)\n", config.Title, config.Version)
	return config
}

func toMap(a []interface{}) []map[string]interface{} {
	r := []map[string]interface{}{}
	for _, o := range a {
		r = append(r, o.(map[string]interface{}))
	}
	return r
}

func getResolution(appConfig *AppConfig, mode string) (int, int) {
	runtimeConfig, ok := appConfig.runtime[mode]
	if ok == false {
		panic("Can't find runtime config")
	}
	resolution, ok := (runtimeConfig.(map[string]interface{}))["resolution"]
	if ok == false {
		panic("Can't find resolution in runtime config")
	}
	resArray := (resolution.([]interface{}))
	return int(resArray[0].(float64)), int(resArray[1].(float64))
}

func (app *App) addFonts(appConfig *AppConfig, gameDir, mode string) {
	runtimeConfig, ok := appConfig.runtime[mode]
	if ok == false {
		panic("Can't find runtime config")
	}
	fonts, ok := (runtimeConfig.(map[string]interface{}))["fonts"]
	if ok == false {
		panic("Can't find fonts")
	}
	for _, fontBlockI := range fonts.([]interface{}) {
		fontBlock := fontBlockI.(map[string]interface{})
		fontSize, ok := fontBlock["fontSize"]
		if ok == false {
			panic("Can't find fontSize in runtime config")
		}
		fontName, ok := fontBlock["font"]
		if ok == false {
			panic("Can't find fontSize in runtime config")
		}
		var alphaMin, alphaDiv uint8
		alphaMinI, ok := fontBlock["alphaMin"]
		if ok {
			alphaMin = uint8(alphaMinI.(float64))
		}
		alphaDivI, ok := fontBlock["alphaDiv"]
		if ok {
			alphaDiv = uint8(alphaDivI.(float64))
		}

		font, err := NewFont(filepath.Join(gameDir, fontName.(string)), int(fontSize.(float64)), alphaMin, alphaDiv)
		if err != nil {
			panic(err)
		}
		app.Fonts = append(app.Fonts, font)
	}
}

func initUserdir(gameName string) string {
	// create user dir if needed
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	dir := filepath.Join(userHomeDir, "."+gameName)
	fmt.Printf("Game state path: %s\n", dir)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.Mkdir(dir, os.ModePerm)
	}
	return dir
}

func initWindow(windowWidth, windowHeight int) *glfw.Window {
	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	window, err := glfw.CreateWindow(windowWidth, windowHeight, "isongn", nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()

	// Initialize Glow
	if err := gl.Init(); err != nil {
		panic(err)
	}

	version := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Println("OpenGL version", version)

	return window
}

func (app *App) IsDown(key glfw.Key) bool {
	_, ok := app.KeyState[key]
	return ok
}

func (app *App) IsDownMod(key glfw.Key, mod glfw.ModifierKey) bool {
	event, ok := app.KeyState[key]
	if ok {
		return event.Mods&mod > 0
	}
	return false
}

func (app *App) IsFirstDown(key glfw.Key) bool {
	event, ok := app.KeyState[key]
	if ok && event.First {
		event.First = false
		return true
	}
	return false
}

func (app *App) IsFirstDownMod(key glfw.Key, mod glfw.ModifierKey) bool {
	event, ok := app.KeyState[key]
	if ok && event.First && event.Mods&mod > 0 {
		event.First = false
		return true
	}
	return false
}

func (app *App) Keypressed(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	if app.Loading {
		return
	}
	if action == glfw.Release {
		delete(app.KeyState, key)
	} else {
		event, ok := app.KeyState[key]
		if ok {
			event.First = false
		} else {
			event = &KeyPress{
				First: true,
			}
		}
		event.Key = key
		event.Scancode = scancode
		event.Action = action
		event.Mods = mods
		app.KeyState[key] = event
	}
}

func (app *App) IsDownAlt1(key1 glfw.Key) bool {
	return app.IsFirstDown(key1) || app.IsDownMod(key1, glfw.ModShift)
}

func (app *App) IsDownAlt(key1, key2 glfw.Key) bool {
	return app.IsDownAlt1(key1) || app.IsDownAlt1(key2)
}

func (app *App) MouseScroll(w *glfw.Window, xoffs, yoffs float64) {
	app.View.Zoom(yoffs)
}

func (app *App) PanelAtMouse() (*Panel, int, int) {
	return app.Ui.PanelAt(int(app.MousePixelX), int(app.MousePixelY))
}

func (app *App) MousePos(w *glfw.Window, xpos float64, ypos float64) {
	app.MouseX = int32(xpos)
	app.MouseY = int32(ypos)
	app.MousePixelX = int32(xpos / float64(app.windowWidth) * float64(app.Width))
	app.MousePixelY = int32(ypos / float64(app.windowHeight) * float64(app.Height))
	if app.cursorPanel != nil {
		app.Ui.MovePanel(app.cursorPanel, int(app.MousePixelX), int(app.MousePixelY))
	}
	if app.DraggedPanel != nil {
		app.Ui.MovePanel(app.DraggedPanel, int(app.MousePixelX)-app.DraggedPanelOffsX, int(app.MousePixelY)-app.DraggedPanelOffsY)
	} else {
		if app.MouseButtonAction == 1 && app.Dragging == false {
			if math.Abs(float64(app.MouseX)-float64(app.DragStartX)) > float64(minDragPixels) || math.Abs(float64(app.MouseY)-float64(app.DragStartY)) > float64(minDragPixels) {
				app.Dragging = true
				pixelX, pixelY := app.toPixelCoords(app.DragStartX, app.DragStartY)
				app.DragAction, app.DragIndex = app.Game.DragFromUi(pixelX, pixelY)
				if app.DragAction != "" {
					// drag from ui
					app.View.SetClick(0, 0, 0)
				} else {
					// drag the ui
					app.DraggedPanel, app.DraggedPanelOffsX, app.DraggedPanelOffsY = app.Ui.PanelAt(pixelX, pixelY)
					if app.DraggedPanel == nil {
						// drag from map
						app.readSelection = readDragPos
					}
				}
			}
		}
	}
}

func (app *App) toPixelCoords(windowX, windowY int32) (int, int) {
	return int(float32(windowX) * float32(app.Width) / float32(app.windowWidth)), int(float32(windowY) * float32(app.Height) / float32(app.windowHeight))
}

func (app *App) MouseClick(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	if action == 1 {
		app.MouseButtonAction = 1
		app.DragStartX = app.MouseX
		app.DragStartY = app.MouseY
	} else {
		app.MouseButtonAction = 0
		if app.DraggedPanel == nil {
			if app.Dragging == false {
				// click on a ui item?
				app.DragAction, app.DragIndex = app.Game.DragFromUi(int(app.MousePixelX), int(app.MousePixelY))
			}
			app.readSelection = readMousePos
		} else {
			app.DraggedPanel = nil
			app.Dragging = false
		}
	}
}

func (app *App) CompleteDrag() {
	if app.MouseButtonAction == 0 {
		app.Dragging = false
	}
}

func (app *App) SetCursorShape(shapeIndex int) {
	app.HideCursorShape()
	shape := shapes.Shapes[shapeIndex]
	w := shape.Image.Bounds().Dx()
	h := shape.Image.Bounds().Dy()
	init := true
	app.cursorPanel = app.Ui.AddBg(int(app.MousePixelX)-w/2, int(app.MousePixelY)-h/2, w, h, color.Transparent, func(panel *Panel) bool {
		if init {
			init = false
			panel.Clear()
			draw.Draw(panel.Rgba, image.Rect(0, 0, w, h), shape.Image, image.Point{0, 0}, draw.Over)
			return true
		}
		return false
	})
}

func (app *App) HideCursorShape() {
	if app.cursorPanel != nil {
		app.Ui.Remove(app.cursorPanel)
		app.cursorPanel = nil
	}
}

func (app *App) CalcFps() {
	currentTime := glfw.GetTime()
	delta := currentTime - app.lastUpdate
	app.nbFrames++
	if delta >= 1.0 { // If last cout was more than 1 sec ago
		app.Window.SetTitle(fmt.Sprintf("%s - %.2f", app.Config.Title, float64(app.nbFrames)/delta))
		app.nbFrames = 0
		app.lastUpdate = currentTime
	}
}

func (app *App) Sleep(lastTime float64) (float64, float64) {
	now := glfw.GetTime()
	d := now - lastTime
	sleep := ((1.0 / app.targetFps) - d) * 1000.0
	if sleep > 0 {
		time.Sleep(time.Duration(sleep) * time.Millisecond)
	}
	return now, d
}

func (app *App) Run() {
	app.Game.Init(app, app.Config.runtime[app.Game.Name()].(map[string]interface{}))

	// Configure global settings
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)
	// gl.ClearColor(0, 0, 0, 0)

	last := glfw.GetTime()
	var delta float64
	selection := [4]byte{}
	mouseVector := mgl32.Vec2{0, 0}
	for !app.Window.ShouldClose() {
		// reduce fan noise / run at target fps
		last, delta = app.Sleep(last)

		// show FPS in window title
		app.CalcFps()

		app.incrFade(last)

		if app.readSelection != dontReadPos {
			// mouse click selection
			app.frameBuffer.Enable(app.Width, app.Height)
			app.View.Draw(delta, true)
			app.frameBuffer.Draw(app.windowWidthDpi, app.windowHeightDpi, app.fade)
			mouseVector[0] = float32(app.MouseX)
			mouseVector[1] = float32(app.MouseY)
			if app.readSelection == readDragPos {
				mouseVector[0] = float32(app.DragStartX)
				mouseVector[1] = float32(app.DragStartY)
			}
			gl.ReadPixels(
				int32(mouseVector[0]*app.dpiX+0.5),
				int32(float32(app.pxHeight)-(mouseVector[1]*app.dpiY+0.5)),
				1, 1,
				gl.RGBA, gl.UNSIGNED_BYTE,
				gl.Ptr(&selection[0]),
			)

			vx := int(selection[0])
			vy := int(selection[1])
			vz := int(selection[2])
			if app.readSelection == readMousePos && app.Dragging {
				// drag drop: find closest position to mouse
				if wx, wy, wz, ok := app.View.GetClosestSurfacePoint(mouseVector, vx, vy, vz, app.windowWidth, app.windowHeight); ok {
					app.View.SetClick(wx, wy, wz)
				}
			} else {
				// click: use exact mouse location
				cancelDrag := false
				if app.Dragging {
					blockPos := app.View.getShapeAt(vx, vy, vz)
					if blockPos != nil && blockPos.pos.Block >= 0 {
						shape := shapes.Shapes[blockPos.pos.Block-1]
						cancelDrag = !shape.IsDraggable
					}
				}
				if cancelDrag {
					app.Dragging = false
				} else {
					wx, wy, wz := app.View.toWorldPos(vx, vy, vz)
					app.View.SetClick(wx, wy, wz)
				}
			}
			app.readSelection = dontReadPos
		}

		// handle events
		app.Game.Events(delta, app.fadeDir, app.MousePixelX, app.MousePixelY)

		app.frameBuffer.Enable(app.Width, app.Height)
		app.View.Draw(delta, false)
		app.frameBuffer.Draw(app.windowWidthDpi, app.windowHeightDpi, app.fade)

		app.uiFrameBuffer.Enable(app.Width, app.Height)
		app.Ui.Draw()
		app.uiFrameBuffer.Draw(app.windowWidthDpi, app.windowHeightDpi, app.fade)

		// Maintenance
		app.Window.SwapBuffers()
		glfw.PollEvents()
	}
}

func (app *App) FadeOut(fx func()) {
	app.fadeDir = -1
	app.fade = 1
	app.fadeTimer = glfw.GetTime() + fadeInterval
	app.fadeFx = fx
}

func (app *App) FadeIn(fx func()) {
	app.fadeDir = 1
	app.fade = 0
	app.fadeTimer = glfw.GetTime() + fadeInterval
	app.fadeFx = fx
}

func (app *App) FadeDone() {
	app.fadeFx = nil
	app.fadeDir = 0
	app.fade = 1
}

func (app *App) incrFade(last float64) {
	if app.fadeFx != nil {
		if app.fadeTimer > last {
			app.fade = float32((app.fadeTimer - last) / fadeInterval)
			if app.fadeDir == 1 {
				app.fade = 1 - app.fade
			}
		} else {
			app.fadeFx()
		}
	}
}
