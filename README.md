# glhf

open**GL** **H**ave **F**un - A Go package that makes life with OpenGL enjoyable.

## Main features

- Garbage collected OpenGL objects
- Dynamically sized vertex slices (vertex arrays are boring)
- Textures, Shaders, Frames (reasonably managed framebuffers)
- Always possible to use standard OpenGL with `glhf`

## Motivation

OpenGL is verbose, it's usage patterns are repetitive and it's manual memory management doesn't fit
Go's design. When developing a game development library, it's usually desirable to create some
higher-level abstractions around OpenGL. This library is a take on that.