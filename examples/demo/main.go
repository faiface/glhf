package main

import (
	"image"
	"image/draw"
	_ "image/png"
	"os"

	"github.com/faiface/glhf"
	"github.com/faiface/mainthread"
	"github.com/go-gl/glfw/v3.1/glfw"
)

func loadImage(path string) (*image.NRGBA, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	nrgba := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(nrgba, nrgba.Bounds(), img, bounds.Min, draw.Src)
	return nrgba, nil
}

func run() {
	var win *glfw.Window

	defer func() {
		mainthread.Call(func() {
			glfw.Terminate()
		})
	}()

	mainthread.Call(func() {
		glfw.Init()

		glfw.WindowHint(glfw.ContextVersionMajor, 3)
		glfw.WindowHint(glfw.ContextVersionMinor, 3)
		glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
		glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
		glfw.WindowHint(glfw.Resizable, glfw.False)

		var err error

		win, err = glfw.CreateWindow(560, 697, "GLHF Rocks!", nil, nil)
		if err != nil {
			panic(err)
		}

		win.MakeContextCurrent()

		glhf.Init()
	})

	var (
		// Here we define a vertex format of our vertex slice. It's actually a basic slice
		// literal.
		//
		// The vertex format consists of names and types of the attributes. The name is the
		// name that the attribute is referenced by inside a shader.
		vertexFormat = glhf.AttrFormat{
			{Name: "position", Type: glhf.Vec2},
			{Name: "texture", Type: glhf.Vec2},
		}

		// Here we declare some variables for later use.
		shader  *glhf.Shader
		texture *glhf.Texture
		slice   *glhf.VertexSlice
	)

	// Here we load an image from a file. The loadImage function is not within the library, it
	// just loads and returns a image.NRGBA.
	gopherImage, err := loadImage("celebrate.png")
	if err != nil {
		panic(err)
	}

	// Every OpenGL call needs to be done inside the main thread.
	mainthread.Call(func() {
		var err error

		// Here we create a shader. The second argument is the format of the uniform
		// attributes. Since our shader has no uniform attributes, the format is empty.
		shader, err = glhf.NewShader(vertexFormat, glhf.AttrFormat{}, vertexShader, fragmentShader)

		// If the shader compilation did not go successfully, an error with a full
		// description is returned.
		if err != nil {
			panic(err)
		}

		// We create a texture from the loaded image.
		texture = glhf.NewTexture(
			gopherImage.Bounds().Dx(),
			gopherImage.Bounds().Dy(),
			true,
			gopherImage.Pix,
		)

		// And finally, we make a vertex slice, which is basically a dynamically sized
		// vertex array. The length of the slice is 6 and the capacity is the same.
		//
		// The slice inherits the vertex format of the supplied shader. Also, it should
		// only be used with that shader.
		slice = glhf.MakeVertexSlice(shader, 6, 6)

		// Before we use a slice, we need to Begin it. The same holds for all objects in
		// GLHF.
		slice.Begin()

		// We assign data to the vertex slice. The values are in the order as in the vertex
		// format of the slice (shader). Each two floats correspond to an attribute of type
		// glhf.Vec2.
		slice.SetVertexData([]float32{
			-1, -1, 0, 1,
			+1, -1, 1, 1,
			+1, +1, 1, 0,

			-1, -1, 0, 1,
			+1, +1, 1, 0,
			-1, +1, 0, 0,
		})

		// When we're done with the slice, we End it.
		slice.End()
	})

	shouldQuit := false
	for !shouldQuit {
		mainthread.Call(func() {
			if win.ShouldClose() {
				shouldQuit = true
			}

			// Clear the window.
			glhf.Clear(1, 1, 1, 1)

			// Here we Begin/End all necessary objects and finally draw the vertex
			// slice.
			shader.Begin()
			texture.Begin()
			slice.Begin()
			slice.Draw()
			slice.End()
			texture.End()
			shader.End()

			win.SwapBuffers()
			glfw.PollEvents()
		})
	}
}

func main() {
	mainthread.Run(run)
}

var vertexShader = `
#version 330 core

in vec2 position;
in vec2 texture;

out vec2 Texture;

void main() {
	gl_Position = vec4(position, 0.0, 1.0);
	Texture = texture;
}
`

var fragmentShader = `
#version 330 core

in vec2 Texture;

out vec4 color;

uniform sampler2D tex;

void main() {
	color = texture(tex, Texture);
}
`
