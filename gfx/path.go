package gfx

import (
	"github.com/uzudil/isongn/shapes"
	"github.com/uzudil/isongn/util"
)

type PathStep [3]int

type PathNode struct {
	f, g, h         int
	visited, closed bool
	fitCalled       bool
	blocker         *BlockPos
	debug           string
	parent          *BlockPos
}

type ViewContext struct {
	isFlying          bool
	isPathing         bool
	usePathThrough    bool
	pathThroughShapes map[*shapes.Shape]bool
	start, end        *BlockPos
	startBox, endBox  *BoundingBox
}

func (view *View) AddPathThroughShape(shape *shapes.Shape) {
	view.context.pathThroughShapes[shape] = true
}

func (view *View) FindPath(sx, sy, sz, ex, ey, ez int, isFlying bool, dSrc, dDst int) []PathStep {
	view.context.isFlying = isFlying
	view.context.isPathing = true
	startViewX, startViewY, startViewZ, startOk := view.toViewPos(sx, sy, sz)
	endViewX, endViewY, endViewZ, endOk := view.toViewPos(ex, ey, ez)
	var steps []PathStep
	if startOk && endOk {
		view.context.start = view.blockPos[startViewX][startViewY][startViewZ]
		view.context.end = view.blockPos[endViewX][endViewY][endViewZ]
		view.context.startBox = &BoundingBox{startViewX, startViewY, startViewZ, dSrc, dSrc, 4}
		view.context.endBox = &BoundingBox{endViewX, endViewY, endViewZ, dDst, dDst, 4}
		// first try w/o doors
		view.context.usePathThrough = false
		steps = view.findPath()
		if steps == nil {
			// try again with doors
			view.context.usePathThrough = true
			steps = view.findPath()
		}
	}
	view.context.isPathing = false
	return steps
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
func (view *View) findPath() []PathStep {
	view.resetPathFind()
	openList := []*BlockPos{view.context.start}
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
		view.context.startBox.SetPos(currentNode.x, currentNode.y, currentNode.z)
		if view.context.startBox.intersect(view.context.endBox) {
			//if currentNode == view.context.end {
			return view.generatePath(currentNode)
		}

		// Normal case -- move currentNode from open to closed, process each of its neighbors
		openList = remove(openList, lowInd)
		currentNode.pathNode.closed = true

		// fmt.Printf("Processing: %d,%d,%d. List len=%d\n", currentNode.x, currentNode.y, currentNode.z, len(openList))

		neighbors := view.astarNeighbors(currentNode)
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
					neighbor.pathNode.h = view.heuristic(neighbor, view.context.end)
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

func (view *View) heuristic(pos0, pos1 *BlockPos) int {
	// Manhattan distance. See list of heuristics: http://theory.stanford.edu/~amitp/GameProgramming/Heuristics.html
	d1 := util.AbsInt(pos1.x + view.context.startBox.W/2 - pos0.x + view.context.endBox.W/2)
	d2 := util.AbsInt(pos1.y + view.context.startBox.W/2 - pos0.y + view.context.endBox.W/2)
	d3 := util.AbsInt(pos1.z + view.context.startBox.W/2 - pos0.z + view.context.endBox.W/2)
	return d1 + d2 + d3
}

func (view *View) astarNeighbors(node *BlockPos) []*BlockPos {
	ret := []*BlockPos{}
	if node.x-1 >= 0 {
		if newNode := view.tryInDir(node, -1, 0); newNode != nil {
			ret = append(ret, newNode)
		}
	}
	if node.x+1 < SIZE {
		if newNode := view.tryInDir(node, 1, 0); newNode != nil {
			ret = append(ret, newNode)
		}
	}
	if node.y-1 >= 0 {
		if newNode := view.tryInDir(node, 0, -1); newNode != nil {
			ret = append(ret, newNode)
		}
	}
	if node.y+1 < SIZE {
		if newNode := view.tryInDir(node, 0, 1); newNode != nil {
			ret = append(ret, newNode)
		}
	}
	return ret
}

func (view *View) tryInDir(node *BlockPos, dx, dy int) *BlockPos {
	return view.tryMove(node.x+dx, node.y+dy, node.z)
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
