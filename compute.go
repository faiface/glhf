package glhf

import (
	"fmt"
	"runtime"

	"github.com/faiface/mainthread"
	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

// Compute is an OpenGL shader program.
type Compute struct {
	program    binder
	vertexFmt  AttrFormat
	uniformFmt AttrFormat
	uniformLoc []int32
}

// NewCompute creates a new shader program from the specified vertex shader and fragment shader
// sources.
//
// Note that vertexShader and fragmentShader parameters must contain the source code, they're
// not filenames.
func NewCompute(vertexFmt, uniformFmt AttrFormat, vertexShader, fragmentShader string) (*Compute, error) {
	shader := &Compute{
		program: binder{
			restoreLoc: gl.CURRENT_PROGRAM,
			bindFunc: func(obj uint32) {
				gl.UseProgram(obj)
			},
		},
		vertexFmt:  vertexFmt,
		uniformFmt: uniformFmt,
		uniformLoc: make([]int32, len(uniformFmt)),
	}

	var vshader, fshader, cshader uint32

	// vertex shader
	{
		vshader = gl.CreateShader(gl.VERTEX_SHADER)
		src, free := gl.Strs(computeVertexShader)
		defer free()
		length := int32(len(computeVertexShader))
		gl.ShaderSource(vshader, 1, src, &length)
		gl.CompileShader(vshader)

		var success int32
		gl.GetShaderiv(vshader, gl.COMPILE_STATUS, &success)
		if success == gl.FALSE {
			var logLen int32
			gl.GetShaderiv(vshader, gl.INFO_LOG_LENGTH, &logLen)

			infoLog := make([]byte, logLen)
			gl.GetShaderInfoLog(vshader, logLen, nil, &infoLog[0])
			return nil, fmt.Errorf("error compiling vertex shader: %s", string(infoLog))
		}

		defer gl.DeleteShader(vshader)
	}

	// fragment shader
	{
		fshader = gl.CreateShader(gl.FRAGMENT_SHADER)
		src, free := gl.Strs(computeFragmentShader)
		defer free()
		length := int32(len(computeFragmentShader))
		gl.ShaderSource(fshader, 1, src, &length)
		gl.CompileShader(fshader)

		var success int32
		gl.GetShaderiv(fshader, gl.COMPILE_STATUS, &success)
		if success == gl.FALSE {
			var logLen int32
			gl.GetShaderiv(fshader, gl.INFO_LOG_LENGTH, &logLen)

			infoLog := make([]byte, logLen)
			gl.GetShaderInfoLog(fshader, logLen, nil, &infoLog[0])
			return nil, fmt.Errorf("error compiling fragment shader: %s", string(infoLog))
		}

		defer gl.DeleteShader(fshader)
	}

	// compute shader
	{
		cshader = gl.CreateShader(gl.COMPUTE_SHADER)
		src, free := gl.Strs(computeShader)
		defer free()
		length := int32(len(computeShader))
		gl.ShaderSource(cshader, 1, src, &length)
		gl.CompileShader(cshader)

		var success int32
		gl.GetShaderiv(cshader, gl.COMPILE_STATUS, &success)
		if success == gl.FALSE {
			var logLen int32
			gl.GetShaderiv(cshader, gl.INFO_LOG_LENGTH, &logLen)

			infoLog := make([]byte, logLen)
			gl.GetShaderInfoLog(cshader, logLen, nil, &infoLog[0])
			return nil, fmt.Errorf("error compiling compute shader: %s", string(infoLog))
		}

		defer gl.DeleteShader(cshader)
	}

	// shader program
	{
		shader.program.obj = gl.CreateProgram()
		gl.AttachShader(shader.program.obj, vshader)
		gl.AttachShader(shader.program.obj, fshader)
		gl.AttachShader(shader.program.obj, cshader)
		gl.LinkProgram(shader.program.obj)

		var success int32
		gl.GetProgramiv(shader.program.obj, gl.LINK_STATUS, &success)
		if success == gl.FALSE {
			var logLen int32
			gl.GetProgramiv(shader.program.obj, gl.INFO_LOG_LENGTH, &logLen)

			infoLog := make([]byte, logLen)
			gl.GetProgramInfoLog(shader.program.obj, logLen, nil, &infoLog[0])
			return nil, fmt.Errorf("error linking shader program: %s", string(infoLog))
		}
	}

	// uniforms
	for i, uniform := range uniformFmt {
		loc := gl.GetUniformLocation(shader.program.obj, gl.Str(uniform.Name+"\x00"))
		shader.uniformLoc[i] = loc
	}

	runtime.SetFinalizer(shader, (*Compute).delete)

	return shader, nil
}

func (s *Compute) delete() {
	mainthread.CallNonBlock(func() {
		gl.DeleteProgram(s.program.obj)
	})
}

// ID returns the OpenGL ID of this Shader.
func (s *Compute) ID() uint32 {
	return s.program.obj
}

// VertexFormat returns the vertex attribute format of this Shader. Do not change it.
func (s *Compute) VertexFormat() AttrFormat {
	return s.vertexFmt
}

// UniformFormat returns the uniform attribute format of this Shader. Do not change it.
func (s *Compute) UniformFormat() AttrFormat {
	return s.uniformFmt
}

var computeShader = `
#version 330 core

#extension GL_ARB_compute_shader : enable
#extension GL_ARB_shader_storage_buffer_object : enable

precision highp sampler2D;

layout( std140, binding=1 ) buffer Pos {
    vec2 pos[];
};

layout( std140, binding=2 ) buffer Vel {
    vec2 vel[];
};

layout(local_size_x = WORK_GROUP_SIZE,  local_size_y = 1, local_size_z = 1) in;

// compute shader to update particles
void main() {
    uint i = gl_GlobalInvocationID.x;
	uint numParticles = 1024;
	float damping = 0.95;
    // thread block size may not be exact multiple of number of particles
    if (i >= numParticles) return;

    // read particle position and velocity from buffers
    vec2 p = pos[i].xy;
    vec2 v = vel[i].xy;

    // integrate
    p += v;
    v *= damping;

    // write new values
    pos[i] = p;
    vel[i] = v;
}
`

var computeVertexShader = `
#version 330 core

in vec2 position;
in vec4 color;
in vec2 texCoords;
in float intensity;

out vec4 Color;
out vec2 texcoords;
out float Intensity;

uniform mat3 u_transform;
uniform vec4 u_bounds;

void main() {
	vec2 transPos = (u_transform * vec3(position, 1.0)).xy;
	vec2 normPos = (transPos - u_bounds.xy) / u_bounds.zw * 2 - vec2(1, 1);
	gl_Position = vec4(normPos, 0.0, 1.0);
	Color = color;
	texcoords = texCoords;
	Intensity = intensity;
}
`

var computeFragmentShader = `
#version 330 core

in vec4 Color;
in vec2 texcoords;
in float Intensity;

out vec4 fragColor;

uniform vec4 u_colormask;
uniform vec4 u_texbounds;
uniform sampler2D u_texture;

void main() {
	if (Intensity == 0) {
		fragColor = u_colormask * Color;
	} else {
		fragColor = vec4(0, 0, 0, 0);
		fragColor += (1 - Intensity) * Color;
		vec2 t = (texcoords - u_texbounds.xy) / u_texbounds.zw;
		fragColor += Intensity * Color * texture(u_texture, t);
		fragColor *= u_colormask;
	}
}
`

// SetUniformAttr sets the value of a uniform attribute of this Shader. The attribute is
// specified by the index in the Shader's uniform format.
//
// If the uniform attribute does not exist in the Shader, this method returns false.
//
// Supplied value must correspond to the type of the attribute. Correct types are these
// (right-hand is the type of the value):
//   Attr{Type: Int}:   int32
//   Attr{Type: Float}: float32
//   Attr{Type: Vec2}:  mgl32.Vec2
//   Attr{Type: Vec3}:  mgl32.Vec3
//   Attr{Type: Vec4}:  mgl32.Vec4
//   Attr{Type: Mat2}:  mgl32.Mat2
//   Attr{Type: Mat23}: mgl32.Mat2x3
//   Attr{Type: Mat24}: mgl32.Mat2x4
//   Attr{Type: Mat3}:  mgl32.Mat3
//   Attr{Type: Mat32}: mgl32.Mat3x2
//   Attr{Type: Mat34}: mgl32.Mat3x4
//   Attr{Type: Mat4}:  mgl32.Mat4
//   Attr{Type: Mat42}: mgl32.Mat4x2
//   Attr{Type: Mat43}: mgl32.Mat4x3
// No other types are supported.
//
// The Shader must be bound before calling this method.
func (s *Compute) SetUniformAttr(uniform int, value interface{}) (ok bool) {
	if s.uniformLoc[uniform] < 0 {
		return false
	}
	switch s.uniformFmt[uniform].Type {
	case Int:
		value := value.(int32)
		gl.Uniform1iv(s.uniformLoc[uniform], 1, &value)
	case Intp:
		value := *value.(*int32)
		gl.Uniform1iv(s.uniformLoc[uniform], 1, &value)
	case Float:
		value := value.(float32)
		gl.Uniform1fv(s.uniformLoc[uniform], 1, &value)
	case Floatp:
		value := *value.(*float32)
		gl.Uniform1fv(s.uniformLoc[uniform], 1, &value)
	case Vec2:
		value := value.(mgl32.Vec2)
		gl.Uniform2fv(s.uniformLoc[uniform], 1, &value[0])
	case Vec2p:
		value := *value.(*mgl32.Vec2)
		gl.Uniform2fv(s.uniformLoc[uniform], 1, &value[0])
	case Vec3:
		value := value.(mgl32.Vec3)
		gl.Uniform3fv(s.uniformLoc[uniform], 1, &value[0])
	case Vec3p:
		value := *value.(*mgl32.Vec3)
		gl.Uniform3fv(s.uniformLoc[uniform], 1, &value[0])
	case Vec4:
		value := value.(mgl32.Vec4)
		gl.Uniform4fv(s.uniformLoc[uniform], 1, &value[0])
	case Vec4p:
		value := *value.(*mgl32.Vec4)
		gl.Uniform4fv(s.uniformLoc[uniform], 1, &value[0])
	case Mat2:
		value := value.(mgl32.Mat2)
		gl.UniformMatrix2fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat2p:
		value := *value.(*mgl32.Mat2)
		gl.UniformMatrix2fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat23:
		value := value.(mgl32.Mat2x3)
		gl.UniformMatrix2x3fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat23p:
		value := *value.(*mgl32.Mat2x3)
		gl.UniformMatrix2x3fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat24:
		value := value.(mgl32.Mat2x4)
		gl.UniformMatrix2x4fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat24p:
		value := *value.(*mgl32.Mat2x4)
		gl.UniformMatrix2x4fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat3:
		value := value.(mgl32.Mat3)
		gl.UniformMatrix3fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat3p:
		value := *value.(*mgl32.Mat3)
		gl.UniformMatrix3fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat32:
		value := value.(mgl32.Mat3x2)
		gl.UniformMatrix3x2fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat32p:
		value := *value.(*mgl32.Mat3x2)
		gl.UniformMatrix3x2fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat34:
		value := value.(mgl32.Mat3x4)
		gl.UniformMatrix3x4fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat34p:
		value := *value.(*mgl32.Mat3x4)
		gl.UniformMatrix3x4fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat4:
		value := value.(mgl32.Mat4)
		gl.UniformMatrix4fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat4p:
		value := *value.(*mgl32.Mat4)
		gl.UniformMatrix4fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat42:
		value := value.(mgl32.Mat4x2)
		gl.UniformMatrix4x2fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat42p:
		value := *value.(*mgl32.Mat4x2)
		gl.UniformMatrix4x2fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat43:
		value := value.(mgl32.Mat4x3)
		gl.UniformMatrix4x3fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat43p:
		value := *value.(*mgl32.Mat4x3)
		gl.UniformMatrix4x3fv(s.uniformLoc[uniform], 1, false, &value[0])
	default:
		panic("set uniform attr: invalid attribute type")
	}
	return true
}

// Begin binds the Shader program. This is necessary before using the Shader.
func (s *Compute) Begin() {
	s.program.bind()
}

// End unbinds the Shader program and restores the previous one.
func (s *Compute) End() {
	s.program.restore()
}
