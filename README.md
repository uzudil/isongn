# isongn (iso-engine)

[![IMAGE ALT TEXT HERE](/images/screen1.png)](https://youtu.be/lR5bW-GWPvs)
Click on image to view [video](https://youtu.be/lR5bW-GWPvs).

## What is it?

isongn is a cross-platform, open-world, isometric, scriptable rendering engine. We take care of disk io, graphics, sound, collision detection and map abstractions. You provide the assets, event handling scripts and the vision. Realize your old-school rpg/action-game dreams with easy-to-use modern technology!

## Features
- High speed, smooth-scrolling, isometric rendering of 2D images
- Completely [configurable](https://github.com/uzudil/enalim/blob/main/config.json#L51) rendering engine
- Map-size bound only by disk space and [golang's int](https://yourbasic.org/golang/max-min-int-uint/)
- Simulated [low-res](https://github.com/uzudil/enalim/blob/main/config.json#L12), old-school video resolution
- Game control from [script](https://github.com/uzudil/bscript) with built-in functions for movement, path-finding, etc.
- [Daylight](https://github.com/uzudil/enalim/blob/main/config.json#L22) color cycle, weather effects
- Shader animations for vegeation, etc
- [Animated shapes](https://github.com/uzudil/enalim/blob/main/config.json#L354) for creatures, etc.
- [Truetype font](https://github.com/uzudil/enalim/blob/main/config.json#L13) and simple ui overlay rendering
- Map editor included (extendable via script)

## Example games
- [Enalim](https://github.com/uzudil/enalim)

## The tech

For graphics, isongn uses opengl. Instead of sorting isometric shapes [using the cpu](https://shaunlebron.github.io/IsometricBlocks/), isongn actually draws in 3d space and lets the gpu hardware sort the shapes in the z buffer. It's the best of both worlds: old school graphics and the power of modern hardware.

isongn is written in Go with minimal dependencies so it should run on all platforms.

For scripting, isongn uses [bscript](https://github.com/uzudil/bscript). The language is [similar](https://github.com/uzudil/benji4000/wiki/LanguageFeatures) to modern JavaScript.

## How to use isongn

You can create games without writing any golang code. With a single config file and your assets in a dir, you're ready to set the retro gaming scene on [fire](https://uzudil.itch.io/the-curse-of-svaltfen)!

Please see the [User Guide](https://github.com/uzudil/isongn/wiki/Isongn-User-Guide) for more info about how to make your own games.

2021 (c) Gabor Torok, MIT License