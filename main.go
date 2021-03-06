// Copyright 2014 The go-gl Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Renders a textured spinning cube using GLFW 3 and OpenGL 4.1 core forward-compatible profile.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/uzudil/isongn/editor"
	"github.com/uzudil/isongn/gfx"
	"github.com/uzudil/isongn/runner"
	"github.com/uzudil/isongn/script"
)

func init() {
	// GLFW event handling must run on the main OS thread
	runtime.LockOSThread()
}

func main() {
	gameDir := flag.String("game", "game", "Location of the game assets directory")
	mode := flag.String("mode", "runner", "Game or Editor mode")
	winWidth := flag.Int("width", 800, "Window width (default: 800)")
	winHeight := flag.Int("height", 600, "Window height (default: 600)")
	x := flag.Int("x", 5000, "Editor start X")
	y := flag.Int("y", 5015, "Editor start Y")
	fps := flag.Float64("fps", 60, "Frames per second")
	flag.Parse()

	if err := glfw.Init(); err != nil {
		log.Fatalln("failed to initialize glfw:", err)
	}
	defer glfw.Terminate()

	editor := editor.NewEditor(*x, *y)
	runner := runner.NewRunner()
	var game gfx.Game
	if *mode == editor.Name() {
		game = editor
	} else if *mode == runner.Name() {
		game = runner
	} else {
		fmt.Println("mode must be 'runner' or 'editor'")
		os.Exit(1)
	}
	script.InitScript()
	app := gfx.NewApp(game, *gameDir, *winWidth, *winHeight, *fps)
	app.Run()
}
