package gfx

import (
	"fmt"
	"image"
	"image/draw"
	"math"

	"github.com/go-gl/gl/all-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/uzudil/isongn/shapes"
	"github.com/uzudil/isongn/util"
	"github.com/uzudil/isongn/world"
)

type Texture struct {
	texture      uint32
	textureIndex int32
}

// Block is a displayed Shape
type Block struct {
	vbo                 uint32
	sizeX, sizeY, sizeZ float32
	shape               *shapes.Shape
	texture             *Texture
	index               int32
}

const EXTRA_SIZE = 8

var ZERO_OFFSET [2]float32

type PathNode struct {
	f, g, h         int
	visited, closed bool
	fitCalled       bool
	blocker         *BlockPos
	debug           string
	parent          *BlockPos
}

// BlockPos is a displayed Shape at a location
type BlockPos struct {
	model                  mgl32.Mat4
	x, y, z                int
	worldX, worldY, worldZ int
	pos                    *world.SectionPosition
	box                    BoundingBox
	block                  *Block
	dir                    shapes.Direction
	animationTimer         float64
	animationType          int
	animationSpeed         float64
	ScrollOffset           [2]float32
	pathNode               PathNode
}

type View struct {
	width, height         int
	Loader                *world.Loader
	projection, camera    mgl32.Mat4
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
	vertAttrib            uint32
	texCoordAttrib        uint32
	textures              map[int]*Texture
	blocks                []*Block
	vao                   uint32
	blockPos              [SIZE][SIZE][world.SECTION_Z_SIZE]*BlockPos
	edges                 [SIZE][SIZE]*BlockPos
	zoom                  float64
	shear                 [3]float32
	Cursor                *BlockPos
	ScrollOffset          [3]float32
	maxZ                  int
	underShape            *shapes.Shape
	daylight              [4]float32
}

const viewSize = 10
const SIZE = 96
const DRAW_SIZE = 48
const SEARCH_SIZE = 16

func getProjection(zoom float32, shear [3]float32) mgl32.Mat4 {
	projection := mgl32.Ortho(-viewSize*zoom*0.95, viewSize*zoom*0.95, -viewSize*zoom*0.95, viewSize*zoom*0.95, -viewSize*zoom*2, viewSize*zoom*2)
	m := mgl32.Ident4()
	m.Set(0, 2, shear[0])
	m.Set(1, 2, shear[1])
	m.Set(2, 1, shear[2])
	projection = m.Mul4(projection)
	return projection
}

func InitView(zoom float64, camera, shear [3]float32, loader *world.Loader) *View {
	// does this have to be called in every file?
	var err error
	if err = gl.Init(); err != nil {
		panic(err)
	}

	view := &View{
		zoom:     zoom,
		shear:    shear,
		Loader:   loader,
		maxZ:     world.SECTION_Z_SIZE,
		daylight: [4]float32{1, 1, 1, 1},
	}
	view.projection = getProjection(float32(view.zoom), view.shear)

	// coordinate system: Z is up
	view.camera = mgl32.LookAtV(mgl32.Vec3{camera[0], camera[1], camera[2]}, mgl32.Vec3{0, 0, 0}, mgl32.Vec3{0, 0, 1})

	// Configure the vertex and fragment shaders
	view.program, err = NewProgram(vertexShader, fragmentShader)
	if err != nil {
		panic(err)
	}

	gl.UseProgram(view.program)
	view.projectionUniform = gl.GetUniformLocation(view.program, gl.Str("projection\x00"))
	view.cameraUniform = gl.GetUniformLocation(view.program, gl.Str("camera\x00"))
	view.modelUniform = gl.GetUniformLocation(view.program, gl.Str("model\x00"))
	view.viewScrollUniform = gl.GetUniformLocation(view.program, gl.Str("viewScroll\x00"))
	view.modelScrollUniform = gl.GetUniformLocation(view.program, gl.Str("modelScroll\x00"))
	view.timeUniform = gl.GetUniformLocation(view.program, gl.Str("time\x00"))
	view.heightUniform = gl.GetUniformLocation(view.program, gl.Str("height\x00"))
	view.uniqueOffsetUniform = gl.GetUniformLocation(view.program, gl.Str("uniqueOffset\x00"))
	view.swayEnabledUniform = gl.GetUniformLocation(view.program, gl.Str("swayEnabled\x00"))
	view.bobEnabledUniform = gl.GetUniformLocation(view.program, gl.Str("bobEnabled\x00"))
	view.breatheEnabledUniform = gl.GetUniformLocation(view.program, gl.Str("breatheEnabled\x00"))
	view.textureUniform = gl.GetUniformLocation(view.program, gl.Str("tex\x00"))
	view.textureOffsetUniform = gl.GetUniformLocation(view.program, gl.Str("textureOffset\x00"))
	view.alphaMinUniform = gl.GetUniformLocation(view.program, gl.Str("alphaMin\x00"))
	view.daylightUniform = gl.GetUniformLocation(view.program, gl.Str("daylight\x00"))
	gl.BindFragDataLocation(view.program, 0, gl.Str("outputColor\x00"))
	view.vertAttrib = uint32(gl.GetAttribLocation(view.program, gl.Str("vert\x00")))
	view.texCoordAttrib = uint32(gl.GetAttribLocation(view.program, gl.Str("vertTexCoord\x00")))

	gl.UniformMatrix4fv(view.projectionUniform, 1, false, &view.projection[0])
	gl.UniformMatrix4fv(view.cameraUniform, 1, false, &view.camera[0])
	gl.Uniform1i(view.textureUniform, 0)

	view.textures = map[int]*Texture{}
	gl.GenVertexArrays(1, &view.vao)

	// create a block for each shape
	view.blocks = []*Block{}
	for index, shape := range shapes.Shapes {
		if shape == nil {
			view.blocks = append(view.blocks, nil)
		} else {
			view.blocks = append(view.blocks, view.newBlock(int32(index), shape))
		}
	}
	fmt.Printf("Created %d blocks.\n", len(view.blocks))

	for x := 0; x < SIZE; x++ {
		for y := 0; y < SIZE; y++ {
			for z := 0; z < world.SECTION_Z_SIZE; z++ {
				model := mgl32.Ident4()

				// translate to position
				model.Set(0, 3, float32(x-SIZE/2))
				model.Set(1, 3, float32(y-SIZE/2))
				model.Set(2, 3, float32(z))

				view.blockPos[x][y][z] = &BlockPos{
					x:     x,
					y:     y,
					z:     z,
					model: model,
				}

				if z == 0 {
					edgeModel := mgl32.Ident4()

					// translate to position
					edgeModel.Set(0, 3, float32(x-SIZE/2))
					edgeModel.Set(1, 3, float32(y-SIZE/2))
					edgeModel.Set(2, 3, float32(z)+0.001)

					view.edges[x][y] = &BlockPos{
						x:             x,
						y:             y,
						z:             z,
						model:         edgeModel,
						animationType: shapes.ANIMATION_MOVE,
					}
				}
			}
		}
	}

	view.Cursor = &BlockPos{
		x:     0,
		y:     0,
		z:     0,
		model: mgl32.Ident4(),
	}

	return view
}

func (view *View) newBlock(index int32, shape *shapes.Shape) *Block {
	b := &Block{
		sizeX: shape.Size[0],
		sizeY: shape.Size[1],
		sizeZ: shape.Size[2],
		shape: shape,
		index: index,
	}

	// Configure the vertex data
	gl.BindVertexArray(view.vao)

	gl.GenBuffers(1, &b.vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, b.vbo)
	verts := b.vertices()
	gl.BufferData(gl.ARRAY_BUFFER, len(verts)*4, gl.Ptr(verts), gl.STATIC_DRAW)

	// load the texture if needed
	tex, ok := view.textures[shape.ImageIndex]
	if ok == false {
		texID, err := loadTexture(shapes.Images[shape.ImageIndex])
		if err != nil {
			panic(err)
		}
		tex = &Texture{
			texture:      texID,
			textureIndex: int32(len(view.textures)),
		}
		view.textures[shape.ImageIndex] = tex
	}
	b.texture = tex

	return b
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

func (b *Block) vertices() []float32 {
	// coord system is: z up, x to left, y to right
	//         z
	//         |
	//         |
	//        / \
	//       /   \
	//      x     y
	w := b.sizeX
	h := b.sizeY
	d := b.sizeZ

	// total width/height of texture
	tx := h + w
	ty := h + d + w

	// fudge factor for edges
	var f float32 = b.shape.Fudge

	points := []float32{
		w, 0, d, f, (w - f) / ty,
		w, 0, 0, f, (w + d) / ty,
		w, h, 0, h / tx, 1 - f,
		0, h, 0, 1 - f, (h + d) / ty,
		0, h, d, 1 - f, (h - f) / ty,
		0, 0, d, w / tx, f,
		w, h, d, h / tx, (w + h - f) / ty,
	}

	// scale and translate tex coords to within larger texture
	for i := 0; i < 7; i++ {
		points[i*5+3] *= b.shape.Tex.TexDim[0]
		points[i*5+3] += b.shape.Tex.TexOffset[0]

		points[i*5+4] *= b.shape.Tex.TexDim[1]
		points[i*5+4] += b.shape.Tex.TexOffset[1]
	}

	left := []int{0, 1, 2, 0, 2, 6}
	right := []int{3, 4, 2, 2, 4, 6}
	top := []int{5, 0, 4, 0, 6, 4}

	v := []float32{}
	for _, side := range [][]int{left, right, top} {
		for _, idx := range side {
			for t := 0; t < 5; t++ {
				v = append(v, points[idx*5+t])
			}
		}
	}
	return v
}

func (view *View) SetMaxZ(z int) {
	view.maxZ = z
}

func (view *View) SetUnderShape(shape *shapes.Shape) {
	view.underShape = shape
}

func (view *View) Load() {
	// load
	view.traverse(func(x, y, z int) {
		worldX, worldY, worldZ := view.toWorldPos(x, y, z)
		blockPos := view.blockPos[x][y][z]

		// reset
		blockPos.ScrollOffset[0] = 0
		blockPos.ScrollOffset[1] = 0
		blockPos.worldX = worldX
		blockPos.worldY = worldY
		blockPos.worldZ = worldZ

		view.setPos(blockPos, view.Loader.GetPos(worldX, worldY, worldZ))
	})
}

func (view *View) setPos(blockPos *BlockPos, sectionPos *world.SectionPosition) {
	blockPos.pos = sectionPos
	if sectionPos.Block > 0 {
		block := view.blocks[sectionPos.Block-1]
		blockPos.model.Set(0, 3, float32(blockPos.x-SIZE/2)+block.shape.Offset[0])
		blockPos.model.Set(1, 3, float32(blockPos.y-SIZE/2)+block.shape.Offset[1])
		blockPos.model.Set(2, 3, float32(blockPos.z)+block.shape.Offset[2])
		blockPos.box.Set(
			blockPos.x, blockPos.y, blockPos.z,
			int(block.sizeX), int(block.sizeY), int(block.sizeZ),
		)
	}
}

func (view *View) toWorldPos(viewX, viewY, viewZ int) (int, int, int) {
	return viewX + (view.Loader.X - SIZE/2), viewY + (view.Loader.Y - SIZE/2), viewZ
}

func (view *View) isValidViewPos(viewX, viewY, viewZ int) bool {
	return !(viewX < 0 || viewX >= SIZE || viewY < 0 || viewY >= SIZE || viewZ < 0 || viewZ >= world.SECTION_Z_SIZE)
}

func (view *View) isVisibleViewPos(viewX, viewY, viewZ int) bool {
	return viewX >= SIZE/2-DRAW_SIZE && viewX < SIZE/2+DRAW_SIZE &&
		viewY >= SIZE/2-DRAW_SIZE && viewY < SIZE/2+DRAW_SIZE &&
		viewZ >= 0
}

func (view *View) toViewPos(worldX, worldY, worldZ int) (int, int, int, bool) {
	viewX := worldX - (view.Loader.X - SIZE/2)
	viewY := worldY - (view.Loader.Y - SIZE/2)
	return viewX, viewY, worldZ, view.isValidViewPos(viewX, viewY, worldZ)
}

func (view *View) toScreenPos(worldX, worldY, worldZ int, viewWidth, viewHeight int) (int, int, bool) {
	if viewX, viewY, viewZ, ok := view.toViewPos(worldX, worldY, worldZ); ok {
		pt := mgl32.Vec4{
			float32(viewX-SIZE/2) - view.ScrollOffset[0],
			float32(viewY-SIZE/2) - view.ScrollOffset[1],
			float32(viewZ) - view.ScrollOffset[2],
			1,
		}
		clipSpace := view.projection.Mul4(view.camera).Mul4x1(pt)
		ndcPos := clipSpace.Mul(1 / clipSpace.W()).Vec3()
		vw := float32(viewWidth)
		vh := float32(viewHeight)
		windowSpace := mgl32.Vec2{
			(ndcPos.X() + 1) / 2 * vw,
			((vh / vw) - ndcPos.Y()) / 2 * vw, // no idea why but this works
		}
		return int(windowSpace.X()), int(windowSpace.Y()), true
	}
	return 0, 0, false
}

func (view *View) InView(worldX, worldY, worldZ int) bool {
	_, _, _, validPos := view.toViewPos(worldX, worldY, worldZ)
	return validPos
}

func (view *View) search(viewX, viewY, viewZ int, fx func(*BlockPos) bool) {
	for x := 0; x < SEARCH_SIZE; x++ {
		for y := 0; y < SEARCH_SIZE; y++ {
			for z := 0; z < SEARCH_SIZE; z++ {
				vx := viewX - x
				vy := viewY - y
				vz := viewZ - z
				if view.isValidViewPos(vx, vy, vz) {
					bp := view.blockPos[vx][vy][vz]
					if bp.pos.Block > 0 && fx(bp) {
						return
					}
				}
			}
		}
	}
}

func (view *View) getShapeAt(viewX, viewY, viewZ int) *BlockPos {
	var res *BlockPos
	view.search(viewX, viewY, viewZ, func(bp *BlockPos) bool {
		if bp.box.isInside(viewX, viewY, viewZ) {
			res = bp
			return true
		}
		return false
	})
	return res
}

func (view *View) GetShape(worldX, worldY, worldZ int) (int, int, int, int, bool) {
	viewX, viewY, viewZ, validPos := view.toViewPos(worldX, worldY, worldZ)
	if !validPos {
		return 0, 0, 0, 0, false
	}
	b := view.getShapeAt(viewX, viewY, viewZ)
	if b == nil || b.pos.Block == 0 {
		return 0, 0, 0, 0, false
	}
	originWorldX, originWorldY, originWorldZ := view.toWorldPos(b.x, b.y, b.z)
	return b.pos.Block - 1, originWorldX, originWorldY, originWorldZ, true
}

func (view *View) GetBlocker(toWorldX, toWorldY, toWorldZ int, fromWorldX, fromWorldY, fromWorldZ int) *BlockPos {
	viewX, viewY, viewZ, validPos := view.toViewPos(fromWorldX, fromWorldY, fromWorldZ)
	if !validPos {
		print("WARN: View.GetBlocker src position invalid\n")
		return nil
	}
	src := view.blockPos[viewX][viewY][viewZ]
	if src.pos.Block == 0 {
		fmt.Printf("WARN: View.GetBlocker src position empty %d,%d,%d\n", viewX, viewY, viewZ)
		return nil
	}

	toViewX, toViewY, toViewZ, validPos := view.toViewPos(toWorldX, toWorldY, toWorldZ)
	if !validPos {
		print("WARN: View.GetBlocker dest position invalid\n")
		return nil
	}

	return view.getBlockerAt(toViewX, toViewY, toViewZ, &src.box, src)
}

func (view *View) getBlockerAt(toViewX, toViewY, toViewZ int, box *BoundingBox, src *BlockPos) *BlockPos {
	oldViewX := box.X
	oldViewY := box.Y
	oldViewZ := box.Z
	// fmt.Printf("src=%d,%d dest=%d,%d\n", src.x, src.y, toViewX, toViewY)
	box.SetPos(toViewX, toViewY, toViewZ)

	var blocker *BlockPos
	view.search(toViewX+box.W, toViewY+box.H, toViewZ+box.D, func(bp *BlockPos) bool {
		if bp != src && bp.box.intersect(box) {
			blocker = bp
			return true
		}
		return false
	})
	box.SetPos(oldViewX, oldViewY, oldViewZ)
	return blocker
}

func (view *View) IsEmpty(toWorldX, toWorldY, toWorldZ int, shape *shapes.Shape) bool {
	viewX, viewY, viewZ, validPos := view.toViewPos(toWorldX, toWorldY, toWorldZ)
	if !validPos {
		return false
	}
	box := &BoundingBox{0, 0, 0, int(shape.Size[0]), int(shape.Size[1]), int(shape.Size[2])}
	return view.getBlockerAt(viewX, viewY, viewZ, box, nil) == nil
}

func (view *View) FindTop(worldX, worldY int, shape *shapes.Shape) int {
	maxZ := 0
	viewX, viewY, _, validPos := view.toViewPos(worldX, worldY, maxZ)
	if validPos {
		box := &BoundingBox{0, 0, 0, int(shape.Size[0]), int(shape.Size[1]), int(shape.Size[2])}
		for z := view.maxZ - 1; z >= 0; z-- {
			box.SetPos(viewX, viewY, z)
			view.search(viewX+box.W, viewY+box.H, z+box.D, func(bp *BlockPos) bool {
				if bp.box.intersect(box) && bp.z < view.maxZ && z+1 > maxZ {
					maxZ = z + 1
				}
				return false
			})
		}
	}
	return maxZ
}

// Move a shape from (worldX, worldY, worldZ) to a new position of (newWorldX, newWorldY).
// Returns the new Z value, or -1 if the shape won't fit.
func (view *View) MoveShape(worldX, worldY, worldZ, newWorldX, newWorldY int, isFlying bool) int {
	newViewX, newViewY, _, validPos := view.toViewPos(newWorldX, newWorldY, 0)
	if !validPos {
		return -1
	}

	// figure out the new Z
	newPos := view.tryMove(newViewX, newViewY, worldZ, worldX, worldY, worldZ, false, isFlying)

	// move
	if newPos != nil {
		blockPos, shapeIndex := view.EraseShapeExact(worldX, worldY, worldZ)
		if blockPos != nil {
			view.SetShape(newWorldX, newWorldY, newPos.z, shapeIndex)
		}
		return newPos.z
	}
	return -1
}

func (view *View) SetShape(worldX, worldY, worldZ int, shapeIndex int) *BlockPos {
	view.Loader.SetShape(worldX, worldY, worldZ, shapeIndex)
	viewX, viewY, viewZ, validPos := view.toViewPos(worldX, worldY, worldZ)
	if validPos {
		bp := view.blockPos[viewX][viewY][viewZ]
		view.setPos(bp, view.Loader.GetPos(worldX, worldY, worldZ))
		return bp
	}
	return nil
}

func (view *View) EraseShapeExact(worldX, worldY, worldZ int) (*BlockPos, int) {
	viewX, viewY, viewZ, validPos := view.toViewPos(worldX, worldY, worldZ)
	if validPos {
		blockPos := view.blockPos[viewX][viewY][viewZ]
		if blockPos.pos.Block > 0 {
			shapeIndex := blockPos.pos.Block - 1
			view.Loader.EraseShape(worldX, worldY, worldZ)
			return blockPos, shapeIndex
		}
	}
	return nil, 0
}

func (view *View) EraseShape(worldX, worldY, worldZ int) (*BlockPos, int) {
	if shapeIndex, ox, oy, oz, hasShape := view.GetShape(worldX, worldY, worldZ); hasShape {
		view.Loader.EraseShape(ox, oy, oz)
		viewX, viewY, viewZ, validPos := view.toViewPos(worldX, worldY, worldZ)
		if validPos {
			return view.blockPos[viewX][viewY][viewZ], shapeIndex
		}
	}

	// sometimes this is called for a shape (creature) no longer in view
	// assume the position is its origin and remove it from the sector
	view.Loader.EraseShape(worldX, worldY, worldZ)
	return nil, 0
}

func (view *View) GetBlockPos(worldX, worldY, worldZ int) *BlockPos {
	viewX, viewY, viewZ, validPos := view.toViewPos(worldX, worldY, worldZ)
	if validPos {
		return view.blockPos[viewX][viewY][viewZ]
	}
	return nil
}

func (view *View) SetOffset(worldX, worldY, worldZ int, dx, dy float32) {
	blockPos := view.GetBlockPos(worldX, worldY, worldZ)
	if blockPos != nil {
		blockPos.ScrollOffset[0] = dx
		blockPos.ScrollOffset[1] = dy
	}
}

func (view *View) SetShapeAnimation(worldX, worldY, worldZ int, animationType int, dir shapes.Direction, animationSpeed float64) {
	blockPos := view.GetBlockPos(worldX, worldY, worldZ)
	if blockPos != nil {
		blockPos.dir = dir
		blockPos.animationType = animationType
		blockPos.animationSpeed = animationSpeed
	}
}

func (view *View) traverse(fx func(x, y, z int)) {
	for x := 0; x < SIZE; x++ {
		for y := 0; y < SIZE; y++ {
			for z := 0; z < world.SECTION_Z_SIZE; z++ {
				fx(x, y, z)
			}
		}
	}
}

func (view *View) traverseForDraw(fx func(x, y, z int)) {
	for x := -DRAW_SIZE / 2; x < DRAW_SIZE/2; x++ {
		for y := -DRAW_SIZE / 2; y < DRAW_SIZE/2; y++ {
			for z := 0; z < world.SECTION_Z_SIZE; z++ {
				fx(x+SIZE/2, y+SIZE/2, z)
			}
		}
	}
}

func (view *View) SetCursor(shapeIndex int, z int) {
	view.Cursor.model.Set(2, 3, float32(z))
	view.Cursor.block = nil
	if shapeIndex >= 0 {
		view.Cursor.block = view.blocks[shapeIndex]
	}
}

func (view *View) HideCursor() {
	view.Cursor.block = nil
}

func (view *View) Scroll(dx, dy, dz float32) {
	view.ScrollOffset[0] = dx
	view.ScrollOffset[1] = dy
	view.ScrollOffset[2] = dz
}

func (view *View) isVisible(blockPos *BlockPos) bool {
	if blockPos.pos != nil {
		// is it below the max Z?
		zOk := blockPos.z < view.maxZ
		if view.underShape != nil && zOk {
			// if it's below the max and undershape is set: only display if under (ie. dungeon)
			return zOk && blockPos.pos.Under > 0 && shapes.Shapes[blockPos.pos.Under-1].Group == view.underShape.Group
		}
		if view.underShape == nil && !zOk {
			// if above the max and undershape is not set: show mountain tops
			return blockPos.pos.Block > 0 && shapes.Shapes[blockPos.pos.Block-1].Group > 0
		}
		return zOk
	}
	return false
}

type DrawState struct {
	init    bool
	texture uint32
	vbo     uint32
	delta   float64
	time    float64
}

var state DrawState = DrawState{}

func (view *View) Draw(delta float64) {
	gl.Enable(gl.DEPTH_TEST)
	gl.ClearColor(0, 0, 0, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Enable(gl.BLEND)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.UseProgram(view.program)
	gl.BindVertexArray(view.vao)
	gl.EnableVertexAttribArray(view.vertAttrib)
	gl.EnableVertexAttribArray(view.texCoordAttrib)
	gl.Uniform3fv(view.viewScrollUniform, 1, &view.ScrollOffset[0])
	gl.Uniform4fv(view.daylightUniform, 1, &view.daylight[0])
	state.delta = delta
	state.time += delta
	state.init = false
	view.traverseForDraw(func(x, y, z int) {
		blockPos := view.blockPos[x][y][z]
		if view.isVisible(blockPos) {
			if blockPos.pos.Block > 0 {
				blockPos.Draw(view, view.blocks[blockPos.pos.Block-1], -1)
			}

			modelZ := blockPos.model.At(2, 3)
			for i := range blockPos.pos.Extras {
				// show extras slightly on top of each other
				blockPos.model.Set(2, 3, modelZ+float32(i)*0.01)
				blockPos.Draw(view, view.blocks[blockPos.pos.Extras[i]], i)
			}

			if blockPos.pos.Edge > 0 {
				blockPos.model.Set(2, 3, float32(z)+0.01)
				blockPos.Draw(view, view.blocks[blockPos.pos.Edge-1], -1)
			}
			blockPos.model.Set(2, 3, modelZ)
		}
	})
	if view.Cursor.block != nil {
		view.Cursor.Draw(view, view.Cursor.block, -1)
	}
}

func (b *BlockPos) Draw(view *View, block *Block, extraIndex int) {
	if !state.init || state.texture != block.texture.texture {
		gl.BindTexture(gl.TEXTURE_2D, block.texture.texture)
		state.texture = block.texture.texture
	}
	if !state.init || state.vbo != block.vbo {
		gl.BindBuffer(gl.ARRAY_BUFFER, block.vbo)
		gl.VertexAttribPointer(view.vertAttrib, 3, gl.FLOAT, false, 5*4, gl.PtrOffset(0))
		gl.VertexAttribPointer(view.texCoordAttrib, 2, gl.FLOAT, false, 5*4, gl.PtrOffset(3*4))
		state.vbo = block.vbo
	}
	gl.UniformMatrix4fv(view.modelUniform, 1, false, &b.model[0])
	if extraIndex == -1 {
		gl.Uniform2fv(view.modelScrollUniform, 1, &b.ScrollOffset[0])
	} else {
		gl.Uniform2fv(view.modelScrollUniform, 1, &ZERO_OFFSET[0])
	}
	gl.Uniform1f(view.alphaMinUniform, block.shape.AlphaMin)
	gl.Uniform1f(view.timeUniform, float32(state.time))
	gl.Uniform1f(view.heightUniform, block.shape.Size[2])
	gl.Uniform1i(view.uniqueOffsetUniform, int32(b.worldX+b.worldY+b.worldZ))
	if block.shape.SwayEnabled {
		gl.Uniform1i(view.swayEnabledUniform, 1)
	} else {
		gl.Uniform1i(view.swayEnabledUniform, 0)
	}
	if block.shape.BobEnabled {
		gl.Uniform1i(view.bobEnabledUniform, 1)
	} else {
		gl.Uniform1i(view.bobEnabledUniform, 0)
	}
	if block.shape.BreatheEnabled {
		gl.Uniform1i(view.breatheEnabledUniform, 1)
	} else {
		gl.Uniform1i(view.breatheEnabledUniform, 0)
	}

	animated := false
	if b.dir != shapes.DIR_NONE {
		if animation, ok := block.shape.Animations[b.animationType]; ok {
			b.incrAnimationStep(animation)
			if steps, ok := animation.Tex[b.dir]; ok {
				gl.Uniform1f(view.textureOffsetUniform, steps[animation.AnimationStep].TexOffset[0])
				animated = true
			}
		}
	}
	if !animated {
		gl.Uniform1f(view.textureOffsetUniform, 0)
	}
	gl.DrawArrays(gl.TRIANGLES, 0, 3*2*3)
	state.init = true
}

func (b *BlockPos) incrAnimationStep(animation *shapes.Animation) {
	b.animationTimer -= state.delta
	if b.animationTimer <= 0 {
		b.animationTimer = b.animationSpeed
		animation.AnimationStep++
	}
	if animation.AnimationStep >= animation.Steps {
		animation.AnimationStep = 0
	}
}

func (view *View) Zoom(zoom float64) {
	view.zoom = math.Min(math.Max(view.zoom-zoom*0.1, 0.35), 16)
	// fmt.Printf("zoom:%f\n", view.zoom)
	view.projection = getProjection(float32(view.zoom), view.shear)
	gl.UseProgram(view.program)
	gl.UniformMatrix4fv(view.projectionUniform, 1, false, &view.projection[0])
}

func (view *View) SetDaylight(r, g, b, a float32) {
	view.daylight[0] = r / 255
	view.daylight[1] = g / 255
	view.daylight[2] = b / 255
	view.daylight[3] = 1
}

type PathStep [3]int

func (view *View) FindPath(sx, sy, sz, ex, ey, ez int, isFlying bool) []PathStep {
	startViewX, startViewY, startViewZ, startOk := view.toViewPos(sx, sy, sz)
	endViewX, endViewY, endViewZ, endOk := view.toViewPos(ex, ey, ez)
	if startOk && endOk {
		return view.findPath(startViewX, startViewY, startViewZ, endViewX, endViewY, endViewZ, sx, sy, sz, isFlying)
	}
	return nil
}

/**
	AStar search

	Implemented from: astar-list.js http://github.com/bgrins/javascript-astar
    MIT License

    ** You should not use this implementation (it is quite slower than the heap implementation) **

    Implements the astar search algorithm in javascript
    Based off the original blog post http://www.briangrinstead.com/blog/astar-search-algorithm-in-javascript
    It has since been replaced with astar.js which uses a Binary Heap and is quite faster, but I am leaving
    it here since it is more strictly following pseudocode for the Astar search
*/
func (view *View) findPath(startViewX, startViewY, startViewZ, endViewX, endViewY, endViewZ, startWorldX, startWorldY, startWorldZ int, isFlying bool) []PathStep {
	view.resetPathFind()
	end := view.blockPos[endViewX][endViewY][endViewZ]
	openList := []*BlockPos{view.blockPos[startViewX][startViewY][startViewZ]}
	for len(openList) > 0 {
		// Grab the lowest f(x) to process next
		lowInd := 0
		for i := range openList {
			if openList[i].pathNode.f < openList[lowInd].pathNode.f {
				lowInd = i
			}
		}

		currentNode := openList[lowInd]

		// End case -- result has been found, return the traced path
		if currentNode.x == end.x && currentNode.y == end.y && currentNode.z == end.z {
			return view.generatePath(currentNode)
		}

		// Normal case -- move currentNode from open to closed, process each of its neighbors
		openList = remove(openList, lowInd)
		currentNode.pathNode.closed = true

		// fmt.Printf("Processing: %d,%d,%d. List len=%d\n", currentNode.x, currentNode.y, currentNode.z, len(openList))

		neighbors := view.astarNeighbors(currentNode, startWorldX, startWorldY, startWorldZ, isFlying)
		for _, neighbor := range neighbors {
			// process only valid nodes
			if !neighbor.pathNode.closed {
				// fmt.Printf("\ttrying %d,%d,%d\n", neighbor.x, neighbor.y, neighbor.z)
				// g score is the shortest distance from start to current node, we need to check if
				//   the path we have arrived at this neighbor is the shortest one we have seen yet
				// adding 1: 1 is the distance from a node to it's neighbor
				gScore := currentNode.pathNode.g + 1
				gScoreIsBest := false

				if !neighbor.pathNode.visited {
					// This the the first time we have arrived at this node, it must be the best
					// Also, we need to take the h (heuristic) score since we haven't done so yet
					gScoreIsBest = true
					neighbor.pathNode.h = heuristic(neighbor, end)
					neighbor.pathNode.visited = true
					openList = append(openList, neighbor)
				} else if gScore < neighbor.pathNode.g {
					// We have already seen the node, but last time it had a worse g (distance from start)
					gScoreIsBest = true
				}

				if gScoreIsBest {
					// Found an optimal (so far) path to this node.  Store info on how we got here and
					//  just how good it really is...
					neighbor.pathNode.parent = currentNode
					neighbor.pathNode.g = gScore
					neighbor.pathNode.f = neighbor.pathNode.g + neighbor.pathNode.h
				}
			}
		}
	}

	// No result was found -- nil signifies failure to find path
	return nil
}

func heuristic(pos0, pos1 *BlockPos) int {
	// Manhattan distance. See list of heuristics: http://theory.stanford.edu/~amitp/GameProgramming/Heuristics.html
	d1 := util.AbsInt(pos1.x - pos0.x)
	d2 := util.AbsInt(pos1.y - pos0.y)
	d3 := util.AbsInt(pos1.z - pos0.z)
	return d1 + d2 + d3
}

func (view *View) astarNeighbors(node *BlockPos, startWorldX, startWorldY, startWorldZ int, isFlying bool) []*BlockPos {
	ret := []*BlockPos{}
	if node.x-1 >= 0 {
		if newNode := view.tryInDir(node, -1, 0, startWorldX, startWorldY, startWorldZ, isFlying); newNode != nil {
			ret = append(ret, newNode)
		}
	}
	if node.x+1 < SIZE {
		if newNode := view.tryInDir(node, 1, 0, startWorldX, startWorldY, startWorldZ, isFlying); newNode != nil {
			ret = append(ret, newNode)
		}
	}
	if node.y-1 >= 0 {
		if newNode := view.tryInDir(node, 0, -1, startWorldX, startWorldY, startWorldZ, isFlying); newNode != nil {
			ret = append(ret, newNode)
		}
	}
	if node.y+1 < SIZE {
		if newNode := view.tryInDir(node, 0, 1, startWorldX, startWorldY, startWorldZ, isFlying); newNode != nil {
			ret = append(ret, newNode)
		}
	}
	return ret
}

func (view *View) tryInDir(node *BlockPos, dx, dy, startWorldX, startWorldY, startWorldZ int, isFlying bool) *BlockPos {
	return view.tryMove(node.x+dx, node.y+dy, node.z, startWorldX, startWorldY, startWorldZ, true, isFlying)
}

func (view *View) tryMove(newViewX, newViewY, newViewZ, startWorldX, startWorldY, startWorldZ int, cacheFit, isFlying bool) *BlockPos {
	// can we drop down here? (check this before the same-z move)
	z := newViewZ
	var standingOn *BlockPos
	for z > 0 {
		standingOn = view.getBlocker(view.blockPos[newViewX][newViewY][z-1], startWorldX, startWorldY, startWorldZ, cacheFit)
		if standingOn != nil {
			break
		}
		z--
	}
	if !isFlying && standingOn != nil && shapes.Shapes[standingOn.pos.Block-1].NoSupport {
		return nil
	}
	if z < newViewZ {
		return view.blockPos[newViewX][newViewY][z]
	}

	// same z move
	newNode := view.blockPos[newViewX][newViewY][newViewZ]
	if view.getBlocker(newNode, startWorldX, startWorldY, startWorldZ, cacheFit) == nil {
		return newNode
	}

	// step up?
	newNode = view.blockPos[newViewX][newViewY][newViewZ+1]
	if view.getBlocker(newNode, startWorldX, startWorldY, startWorldZ, cacheFit) == nil {
		return newNode
	}
	return nil
}

func (view *View) getBlocker(node *BlockPos, startWorldX, startWorldY, startWorldZ int, cacheFit bool) *BlockPos {
	if cacheFit {
		if !node.pathNode.fitCalled {
			node.pathNode.fitCalled = true
			node.pathNode.blocker = view.GetBlocker(
				node.worldX, node.worldY, node.worldZ,
				startWorldX, startWorldY, startWorldZ)
		}
		return node.pathNode.blocker
	} else {
		return view.GetBlocker(
			node.worldX, node.worldY, node.worldZ,
			startWorldX, startWorldY, startWorldZ)
	}
}

func (view *View) generatePath(currentNode *BlockPos) []PathStep {
	ret := []PathStep{}
	for currentNode.pathNode.parent != nil {
		wx, wy, wz := view.toWorldPos(currentNode.x, currentNode.y, currentNode.z)
		ret = append(ret, PathStep{wx, wy, wz})
		currentNode = currentNode.pathNode.parent
	}
	return reverse(ret)
}

func remove(s []*BlockPos, i int) []*BlockPos {
	s[i] = s[len(s)-1]
	// We do not need to put s[i] at the end, as it will be discarded anyway
	return s[:len(s)-1]
}

func reverse(nodes []PathStep) []PathStep {
	for i := 0; i < len(nodes)/2; i++ {
		j := len(nodes) - i - 1
		nodes[i], nodes[j] = nodes[j], nodes[i]
	}
	return nodes
}

func (view *View) resetPathFind() {
	for x := range view.blockPos {
		for y := range view.blockPos[x] {
			for _, blockPos := range view.blockPos[x][y] {
				blockPos.pathNode.f = 0
				blockPos.pathNode.g = 0
				blockPos.pathNode.h = 0
				blockPos.pathNode.blocker = nil
				blockPos.pathNode.fitCalled = false
				blockPos.pathNode.visited = false
				blockPos.pathNode.closed = false
				blockPos.pathNode.parent = nil
				blockPos.pathNode.debug = ""
			}
		}
	}
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
