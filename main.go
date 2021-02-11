package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"

	"github.com/ungerik/go3d/vec3"
)

var maxRayDepth int = 10

type sphere struct {
	centre                       vec3.T
	radius, radius2              float64
	surfaceColor, emissionsColor vec3.T
	reflection, transparency     float64
}

func (s *sphere) intersect(rayorig, raydir *vec3.T) (bol bool, t0 float64, t1 float64) {
	l := vec3.Sub(&s.centre, rayorig)
	tca := float64(vec3.Dot(&l, raydir))
	if tca < 0 {
		bol = false
		return
	}
	var d2 float64 = float64(vec3.Dot(&l, &l)) - tca*tca
	if d2 > s.radius2 {
		bol = false
		return
	}
	thc := math.Sqrt(s.radius2 - float64(d2))
	t0 = tca - thc
	t1 = tca + thc
	bol = true
	return

}
func mix(a, b, mix float64) float64 {
	return (b * mix) + (a * (1 - mix))
}
func mulF(v vec3.T, f float64) *vec3.T {
	temp := vec3.T{v[0] * float32(f), v[1] * float32(f), v[2] * float32(f)}
	return &temp
}
func makeSphere(centre vec3.T, radius float64,
	surfaceColor vec3.T,
	reflection, transparency float64) sphere {
	return sphere{centre, radius, radius * radius, surfaceColor, vec3.Zero, reflection, transparency}
}
func makeSphereEmis(centre vec3.T, radius float64,
	surfaceColor, emissionsColor vec3.T,
	reflection, transparency float64) sphere {
	return sphere{centre, radius, radius * radius, surfaceColor, emissionsColor, reflection, transparency}
}
func trace(rayorig, raydir vec3.T, spheres []sphere, depth int) vec3.T {
	tnear := math.Inf(1)
	//const Sphere* sphere = NULL;
	var sph *sphere
	for i := 0; i < len(spheres); i++ {
		inter, t0, t1 := spheres[i].intersect(&rayorig, &raydir)
		if inter {
			if t0 < 0 {
				t0 = t1
			}
			if t0 < tnear {
				tnear = t0
				sph = &spheres[i]
			}
		}
	}
	if sph == nil {
		return vec3.T{2, 2, 2}

	}

	surfaceColor := vec3.Zero
	phit := vec3.Add(&rayorig, mulF(raydir, tnear))
	nhit := vec3.Sub(&phit, &(*sph).centre)
	nhit.Normalize()
	bias := 1e-4
	inside := false
	if vec3.Dot(&raydir, &nhit) > 0 {
		nhit = nhit.Inverted()
		inside = true
	}
	if (sph.transparency > 0 || sph.reflection > 0) && depth < maxRayDepth {
		facingratio := 0 - float64(vec3.Dot(&raydir, &nhit))
		fresneleffect := mix(math.Pow(1-facingratio, 3), 1, 0.1)
		refldir := vec3.Sub(&raydir, mulF(nhit, float64(2*vec3.Dot(&raydir, &nhit)))) //mulF(vec3.Sub(&raydir, &nhit), float64(2*vec3.Dot(&raydir, &nhit)))
		org := vec3.Add(&phit, mulF(nhit, bias))
		reflection := trace(org, refldir, spheres, depth+1) //vec3.T{0, 1.74, 2}
		refraction := vec3.Zero
		if sph.transparency > 0 {
			ior := 1.1
			eta := ior
			if !inside {
				eta = 1 / ior
			}
			cosi := float64(-vec3.Dot(&nhit, &raydir))
			k := 1 - eta*eta*(1-cosi*cosi)
			refrdir := vec3.Add(mulF(raydir, eta), mulF(nhit, (eta*cosi-math.Sqrt(k))))
			refrdir.Normalize()
			refraction = trace(vec3.Sub(&phit, mulF(nhit, bias)), refrdir, spheres, depth+1)
		}
		temp := vec3.Add(mulF(reflection, fresneleffect), mulF(refraction, (1-fresneleffect)*sph.transparency))
		surfaceColor = vec3.Mul(
			&temp, &sph.surfaceColor)
	}

	return surfaceColor
}
func render(spheres []sphere, iteration int) {

	width, height := 1920, 1080 //640, 480
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	imageg := make([]vec3.T, width*height)
	pixel := 0
	invWidth := 1 / float64(width)
	invHeight := 1 / float64(height)
	var fov float64 = 30
	aspectratio := float64(width) / float64(height)
	angle := math.Tan(math.Pi * 0.5 * fov / 180.)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			xx := (2*((float64(x)+0.5)*invWidth) - 1) * angle * aspectratio
			yy := (1 - 2*((float64(y)+0.5)*invHeight)) * angle
			rayDir := vec3.T{float32(xx), float32(yy), -1}
			rayDir.Normalize()
			imageg[pixel] = trace(vec3.Zero, rayDir, spheres, 0)

			img.Set(x, y,
				color.RGBA{uint8(math.Min(float64(1), float64(imageg[pixel][0])) * 255),
					uint8(math.Min(float64(1), float64(imageg[pixel][1])) * 255),
					uint8(math.Min(float64(1), float64(imageg[pixel][2])) * 255), 255})
			pixel++
		}
	}
	f, err := os.Create("draw.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	png.Encode(f, img)
	//fmt.Printf("P6\n%v %v\n255\n", width, height)
	//for i := 0; i < width*height; i++ {
	//	img.set()
	//fmt.Printf("%c%c%c", int((math.Min(float64(1), float64(image[i][0])) * 255)), int((math.Min(float64(1), float64(image[i][1])) * 255)), int((math.Min(float64(1), float64(image[i][2])) * 255)))
	//	fmt.Printf("%x%x%x", "00DD", "00DD", "00DD")

	//}
	//fmt.Printf("%v", image)
	//fmt.Println(image)
}
func main() {
	var spheres []sphere

	spheres = append(spheres, makeSphereEmis(vec3.T{0.0, -10004, -10}, 10000, vec3.T{0.20, 0.20, 0.}, vec3.T{0.20, 0.20, 0.}, 0, 0))
	spheres = append(spheres, makeSphere(vec3.T{0.0, 0, -20}, 4, vec3.T{1.00, 0.32, 0.36}, 1, 0))
	spheres = append(spheres, makeSphere(vec3.T{5.0, -1, -15}, 2, vec3.T{0.90, 0.76, 0.46}, 1, 0))
	spheres = append(spheres, makeSphere(vec3.T{0.0, 0, -10}, 1, vec3.T{1, 1, 1}, 1, 1))

	render(spheres, 0)
	//fmt.Printf("%v", spheres)
}
