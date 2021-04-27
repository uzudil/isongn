package gfx

import (
	"fmt"
	"strings"

	"github.com/go-gl/gl/all-core/gl"
)

func NewProgram(vertexShaderSource, fragmentShaderSource string) (uint32, error) {
	vertexShader, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
	}

	fragmentShader, err := compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		return 0, err
	}

	program := gl.CreateProgram()

	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to link program: %v", log)
	}

	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)

	return program, nil
}

func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to compile %v: %v", source, log)
	}

	return shader, nil
}

type ViewShader struct {
	program               uint32
	projectionUniform     int32
	cameraUniform         int32
	modelUniform          int32
	textureUniform        int32
	textureOffsetUniform  int32
	alphaMinUniform       int32
	daylightUniform       int32
	viewScrollUniform     int32
	modelScrollUniform    int32
	timeUniform           int32
	heightUniform         int32
	uniqueOffsetUniform   int32
	swayEnabledUniform    int32
	bobEnabledUniform     int32
	breatheEnabledUniform int32
	selectModeUniform     int32
	vertAttrib            uint32
	texCoordAttrib        uint32
}

func (view *View) initShaders() {
	view.shaders = view.initProgram(vertexShader, fragmentShader)
	view.selectShaders = view.initProgram(vertexShader, selectFragmentShader)
}

func (view *View) initProgram(vertexShader, fragmentShader string) *ViewShader {
	vs := &ViewShader{}
	var err error
	vs.program, err = NewProgram(vertexShader, fragmentShader)
	if err != nil {
		panic(err)
	}

	gl.UseProgram(vs.program)
	vs.projectionUniform = gl.GetUniformLocation(vs.program, gl.Str("projection\x00"))
	vs.cameraUniform = gl.GetUniformLocation(vs.program, gl.Str("camera\x00"))
	vs.modelUniform = gl.GetUniformLocation(vs.program, gl.Str("model\x00"))
	vs.viewScrollUniform = gl.GetUniformLocation(vs.program, gl.Str("viewScroll\x00"))
	vs.modelScrollUniform = gl.GetUniformLocation(vs.program, gl.Str("modelScroll\x00"))
	vs.timeUniform = gl.GetUniformLocation(vs.program, gl.Str("time\x00"))
	vs.heightUniform = gl.GetUniformLocation(vs.program, gl.Str("height\x00"))
	vs.uniqueOffsetUniform = gl.GetUniformLocation(vs.program, gl.Str("uniqueOffset\x00"))
	vs.swayEnabledUniform = gl.GetUniformLocation(vs.program, gl.Str("swayEnabled\x00"))
	vs.bobEnabledUniform = gl.GetUniformLocation(vs.program, gl.Str("bobEnabled\x00"))
	vs.breatheEnabledUniform = gl.GetUniformLocation(vs.program, gl.Str("breatheEnabled\x00"))
	vs.textureUniform = gl.GetUniformLocation(vs.program, gl.Str("tex\x00"))
	vs.textureOffsetUniform = gl.GetUniformLocation(vs.program, gl.Str("textureOffset\x00"))
	vs.alphaMinUniform = gl.GetUniformLocation(vs.program, gl.Str("alphaMin\x00"))
	vs.daylightUniform = gl.GetUniformLocation(vs.program, gl.Str("daylight\x00"))
	vs.selectModeUniform = gl.GetUniformLocation(vs.program, gl.Str("selectMode\x00"))
	gl.BindFragDataLocation(vs.program, 0, gl.Str("outputColor\x00"))
	vs.vertAttrib = uint32(gl.GetAttribLocation(vs.program, gl.Str("vert\x00")))
	vs.texCoordAttrib = uint32(gl.GetAttribLocation(vs.program, gl.Str("vertTexCoord\x00")))

	gl.UniformMatrix4fv(vs.projectionUniform, 1, false, &view.projection[0])
	gl.UniformMatrix4fv(vs.cameraUniform, 1, false, &view.camera[0])
	gl.Uniform1i(vs.textureUniform, 0)

	return vs
}

var vertexShader = `
#version 330
uniform mat4 projection;
uniform mat4 camera;
uniform mat4 model;
uniform float textureOffset;
uniform vec3 viewScroll;
uniform vec2 modelScroll;
uniform float time;
uniform float height;
uniform int swayEnabled;
uniform int bobEnabled;
uniform int breatheEnabled;
uniform int uniqueOffset;
in vec3 vert;
in vec2 vertTexCoord;
out vec2 fragTexCoord;
void main() {
    fragTexCoord = vec2(vertTexCoord.x + textureOffset, vertTexCoord.y);

	float swayX = 0;
	if(swayEnabled == 1) {
		swayX = (vert.z / height) * sin(time + uniqueOffset) / 10.0;
	}
	float swayY = 0;
	if(swayEnabled == 1) {
		swayY = (vert.z / height) * cos(time + uniqueOffset) / 10.0;
	}
	float bobZ = 0;
	if(bobEnabled == 1) {
		bobZ = cos((time + uniqueOffset) * 5.0) / 10.0;
	}	
	if(breatheEnabled == 1) {
		bobZ = (vert.z / height) * cos((time + uniqueOffset) * 2.5) / 20.0;
	}	
	float offsX = modelScroll.x - viewScroll.x + swayX;
	float offsY = modelScroll.y - viewScroll.y + swayY;
	float offsZ = bobZ - viewScroll.z;

	// matrix constructor is in column first order
	mat4 modelScroll = mat4(
		1.0, 0.0, 0.0, 0.0,
		0.0, 1.0, 0.0, 0.0,
		0.0, 0.0, 1.0, 0.0,
		model[3][0] + offsX, model[3][1] + offsY, model[3][2] + offsZ, 1.0
	);
    gl_Position = projection * camera * modelScroll * vec4(vert, 1);
}
` + "\x00"

var fragmentShader = `
#version 330
uniform sampler2D tex;
uniform float alphaMin;
uniform vec4 daylight;
uniform vec3 selectMode;
in vec2 fragTexCoord;
layout(location = 0) out vec4 outputColor;
void main() {
	vec4 val = texture(tex, fragTexCoord);
	if (val.a < alphaMin) {
		discard;
	}
	outputColor = val * daylight;
}
` + "\x00"

var selectFragmentShader = `
#version 330
uniform sampler2D tex;
uniform float alphaMin;
uniform vec4 daylight;
uniform vec3 selectMode;
in vec2 fragTexCoord;
layout(location = 0) out vec4 outputColor;
void main() {
	outputColor = vec4(selectMode.r, selectMode.g, selectMode.b, 1);
}
` + "\x00"
