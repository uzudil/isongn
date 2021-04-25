package gfx

import (
	"github.com/go-gl/gl/all-core/gl"
	"github.com/uzudil/isongn/shapes"
)

// Block is a displayed Shape
type Block struct {
	vbo                 uint32
	sizeX, sizeY, sizeZ float32
	shape               *shapes.Shape
	texture             *Texture
	index               int32
}

func (view *View) initBlocks() []*Block {
	var blocks []*Block = []*Block{}
	for index, shape := range shapes.Shapes {
		var block *Block
		if shape != nil {
			block = view.newBlock(int32(index), shape)
		}
		blocks = append(blocks, block)
	}
	return blocks
}

func (view *View) newBlock(index int32, shape *shapes.Shape) *Block {
	b := &Block{
		sizeX:   shape.Size[0],
		sizeY:   shape.Size[1],
		sizeZ:   shape.Size[2],
		shape:   shape,
		index:   index,
		texture: LoadTexture(shape.ImageIndex),
	}

	// Configure the vertex data
	gl.BindVertexArray(view.vao)

	gl.GenBuffers(1, &b.vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, b.vbo)
	verts := b.vertices()
	gl.BufferData(gl.ARRAY_BUFFER, len(verts)*4, gl.Ptr(verts), gl.STATIC_DRAW)

	return b
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
