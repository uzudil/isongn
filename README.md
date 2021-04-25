# isongn (iso-engine)

![alt text](images/anim.gif "Title")

## What is it?

isongn is a cross-platform, open-world, isometric, scriptable rendering engine. We take care of disk io, graphics, sound, collision detection and map abstractions. You provide the assets, event handling scripts and the vision. Realize your old-school rpg/action-game dreams with easy-to-use modern technology!

## Features
- High speed, isometric rendering
- Completely configurable rendering engine
- Simulated low-res, old-school game feel
- Game control from [script](https://github.com/uzudil/bscript)
- Near-infinite map size with 
- Daylight color cycle, weather effects
- Shader animations for vegeation, etc

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