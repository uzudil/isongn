package gfx

import (
	"image"
	"image/color"
	"image/draw"
	"log"

	"github.com/go-gl/gl/all-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

type Ui struct {
	program                   uint32
	modelUniform              int32
	textureUniform            int32
	vertAttrib                uint32
	texCoordAttrib            uint32
	ScreenWidth, ScreenHeight float32
	vao                       uint32
	panels                    []*Panel
}

type Panel struct {
	X, Y, W, H int
	Background color.Color
	Visible    bool
	Rgba       *image.RGBA
	texture    uint32
	model      mgl32.Mat4
	vbo        uint32
	renderer   func(*Panel) bool
	Draggable  bool
}

func InitUi(screenWidth, screenHeight int) *Ui {
	// Initialize Glow
	if err := gl.Init(); err != nil {
		panic(err)
	}

	ui := &Ui{
		ScreenWidth:  float32(screenWidth),
		ScreenHeight: float32(screenHeight),
		panels:       []*Panel{},
	}
	program, err := NewProgram(uiVertexShader, uiFragmentShader)
	if err != nil {
		panic(err)
	}
	ui.program = program

	gl.UseProgram(program)
	ui.modelUniform = gl.GetUniformLocation(program, gl.Str("model\x00"))
	ui.textureUniform = gl.GetUniformLocation(program, gl.Str("tex\x00"))
	gl.BindFragDataLocation(program, 0, gl.Str("outputColor\x00"))
	ui.vertAttrib = uint32(gl.GetAttribLocation(program, gl.Str("vert\x00")))
	ui.texCoordAttrib = uint32(gl.GetAttribLocation(program, gl.Str("vertTexCoord\x00")))

	gl.GenVertexArrays(1, &ui.vao)

	return ui
}

func (ui *Ui) Add(x, y, w, h int, renderer func(*Panel) bool) *Panel {
	return ui.AddBg(x, y, w, h, color.RGBA{0x80, 0x80, 0x80, 0xff}, renderer)
}

func (ui *Ui) AddBg(x, y, w, h int, bg color.Color, renderer func(*Panel) bool) *Panel {
	panel := ui.NewPanel(x, y, w, h, bg, renderer)
	ui.panels = append(ui.panels, panel)
	return panel
}

func (ui *Ui) Remove(panel *Panel) {
	gl.DeleteTextures(1, &panel.texture)
	gl.DeleteBuffers(1, &panel.vbo)
	for index, p := range ui.panels {
		if p == panel {
			ui.panels = append(ui.panels[:index], ui.panels[index+1:]...)
			return
		}
	}
	print("Couldn't find panel in ui list!\n")
}

func (ui *Ui) Raise(panel *Panel) {
	for index, p := range ui.panels {
		if p == panel && index < len(ui.panels)-1 {
			ui.panels[index], ui.panels[len(ui.panels)-1] = ui.panels[len(ui.panels)-1], ui.panels[index]
			return
		}
	}
}

func (ui *Ui) GetTop() *Panel {
	if len(ui.panels) > 0 {
		return ui.panels[len(ui.panels)-1]
	}
	return nil
}

func (ui *Ui) PanelAt(x, y int) (*Panel, int, int) {
	for i := len(ui.panels) - 1; i >= 0; i-- {
		p := ui.panels[i]
		if p.Draggable == false {
			continue
		}
		if p.X <= x && p.X+p.W > x && p.Y <= y && p.Y+p.H > y {
			return p, x - p.X, y - p.Y
		}
	}
	return nil, 0, 0
}

func (ui *Ui) NewPanel(x, y, w, h int, bg color.Color, renderer func(*Panel) bool) *Panel {
	panel := &Panel{
		X:          x,
		Y:          y,
		W:          w,
		H:          h,
		Background: bg,
		Visible:    true,
		renderer:   renderer,
		Draggable:  false,
	}

	panel.Rgba = image.NewRGBA(image.Rect(0, 0, w, h))
	if panel.Rgba.Stride != panel.Rgba.Rect.Size().X*4 {
		log.Fatal("unsupported stride")
	}
	panel.Clear()

	// create a texture
	gl.GenTextures(1, &panel.texture)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, panel.texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)

	// copy image to gpu
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(panel.Rgba.Rect.Size().X),
		int32(panel.Rgba.Rect.Size().Y),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(panel.Rgba.Pix))

	ui.calcModel(panel)

	gl.BindVertexArray(ui.vao)

	gl.GenBuffers(1, &panel.vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, panel.vbo)
	verts := []float32{
		-1, -1, 0, 0, 1,
		-1, 1, 0, 0, 0,
		1, 1, 0, 1, 0,
		1, 1, 0, 1, 0,
		1, -1, 0, 1, 1,
		-1, -1, 0, 0, 1,
	}
	gl.BufferData(gl.ARRAY_BUFFER, len(verts)*4, gl.Ptr(verts), gl.STATIC_DRAW)

	return panel
}

func (ui *Ui) calcModel(panel *Panel) {
	fx := float32(panel.X)
	fy := float32(panel.Y)
	fw := float32(panel.W)
	fh := float32(panel.H)
	panel.model = mgl32.Ident4()
	// translate to position
	panel.model.Set(0, 3, (fx*2-(ui.ScreenWidth-fw))/ui.ScreenWidth)
	panel.model.Set(1, 3, ((ui.ScreenHeight-fh)-fy*2)/ui.ScreenHeight)
	panel.model.Set(2, 3, 0)
	// scale
	panel.model.Set(0, 0, fw/ui.ScreenWidth)
	panel.model.Set(1, 1, fh/ui.ScreenHeight)
	panel.model.Set(2, 2, 1)
}

func (ui *Ui) MovePanel(panel *Panel, x, y int) {
	panel.X = x
	panel.Y = y
	ui.calcModel(panel)
}

func (p *Panel) Clear() {
	// fill with background color
	draw.Draw(p.Rgba, p.Rgba.Bounds(), &image.Uniform{p.Background}, image.ZP, draw.Src)
}

func (ui *Ui) Draw() {
	gl.Disable(gl.DEPTH_TEST)
	gl.ClearColor(0, 0, 0, 0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	// gl.Disable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Enable(gl.BLEND)
	gl.BindVertexArray(ui.vao)
	gl.EnableVertexAttribArray(ui.vertAttrib)
	gl.EnableVertexAttribArray(ui.texCoordAttrib)
	gl.UseProgram(ui.program)
	gl.Uniform1i(ui.textureUniform, 0)
	for _, panel := range ui.panels {
		panel.Draw(ui)
	}
}

func (p *Panel) Draw(ui *Ui) {
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, p.texture)

	// update the texture if needed
	if p.renderer != nil {
		updated := p.renderer(p)
		if updated {
			gl.TexImage2D(
				gl.TEXTURE_2D,
				0,
				gl.RGBA,
				int32(p.Rgba.Rect.Size().X),
				int32(p.Rgba.Rect.Size().Y),
				0,
				gl.RGBA,
				gl.UNSIGNED_BYTE,
				gl.Ptr(p.Rgba.Pix))
		}
	}

	// render
	gl.BindBuffer(gl.ARRAY_BUFFER, p.vbo)
	gl.VertexAttribPointer(ui.vertAttrib, 3, gl.FLOAT, false, 5*4, gl.PtrOffset(0))
	gl.VertexAttribPointer(ui.texCoordAttrib, 2, gl.FLOAT, false, 5*4, gl.PtrOffset(3*4))
	gl.UniformMatrix4fv(ui.modelUniform, 1, false, &p.model[0])
	gl.DrawArrays(gl.TRIANGLES, 0, 3*2*3)
}

var uiVertexShader = `
#version 330
uniform mat4 model;
in vec3 vert;
in vec2 vertTexCoord;
out vec2 fragTexCoord;
void main() {
	fragTexCoord = vertTexCoord;
	// gl_Position = vec4(vert.x, vert.y, vert.z, 1.0);
    gl_Position = model * vec4(vert, 1);
}
` + "\x00"

var uiFragmentShader = `
#version 330
uniform sampler2D tex;
in vec2 fragTexCoord;
layout(location = 0) out vec4 outputColor;
void main() {
	vec4 val = texture(tex, fragTexCoord);
	outputColor = val;
}
` + "\x00"
