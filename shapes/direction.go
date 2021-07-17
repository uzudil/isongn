package shapes

import (
	"math"
)

type Direction int

var DIR_W = Direction(0)
var DIR_SW = Direction(1)
var DIR_S = Direction(2)
var DIR_SE = Direction(3)
var DIR_E = Direction(4)
var DIR_NE = Direction(5)
var DIR_N = Direction(6)
var DIR_NW = Direction(7)
var DIR_NONE = Direction(8)

var Directions = map[string]Direction{
	"w":  DIR_W,
	"sw": DIR_SW,
	"s":  DIR_S,
	"se": DIR_SE,
	"e":  DIR_E,
	"ne": DIR_NE,
	"n":  DIR_N,
	"nw": DIR_NW,
	"":   DIR_NONE,
}

func GetDirScreen(dx, dy float64) Direction {
	r := math.Atan2(dy, dx)
	a := r * 180 / math.Pi
	if a < 0 {
		a += 360
	}
	if a >= 360 {
		a -= 360
	}
	if a >= 30 && a < 60 {
		return DIR_SE
	}
	if a >= 60 && a < 120 {
		return DIR_S
	}
	if a >= 120 && a < 150 {
		return DIR_SW
	}
	if a >= 150 && a < 210 {
		return DIR_W
	}
	if a >= 210 && a < 240 {
		return DIR_NW
	}
	if a >= 240 && a < 300 {
		return DIR_N
	}
	if a >= 300 && a < 330 {
		return DIR_NE
	}
	return DIR_E
}

func GetDir(dx, dy int) Direction {
	if dx == 1 && dy == 0 {
		return DIR_W
	}
	if dx == -1 && dy == 0 {
		return DIR_E
	}
	if dx == 0 && dy == 1 {
		return DIR_S
	}
	if dx == 0 && dy == -1 {
		return DIR_N
	}
	if dx == 1 && dy == 1 {
		return DIR_SW
	}
	if dx == -1 && dy == -1 {
		return DIR_NE
	}
	if dx == -1 && dy == 1 {
		return DIR_SE
	}
	if dx == 1 && dy == -1 {
		return DIR_NW
	}
	return DIR_NONE
}

func (dir Direction) GetDelta() (int, int) {
	switch dir {
	case DIR_W:
		return 1, 0
	case DIR_E:
		return -1, 0
	case DIR_S:
		return 0, 1
	case DIR_N:
		return 0, -1
	case DIR_SW:
		return 1, 1
	case DIR_NE:
		return -1, -1
	case DIR_SE:
		return -1, 1
	case DIR_NW:
		return 1, -1
	default:
		return 0, 0
	}
}
