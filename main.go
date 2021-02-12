package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"log"
	"math"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"

	"github.com/ungerik/go3d/vec3"
)

var maxRayDepth int = 10
var wg sync.WaitGroup

type sphere struct {
	centre                       vec3.T
	radius, radius2              float64
	surfaceColor, emissionsColor vec3.T
	reflection, transparency     float64
}
type cam struct {
	width, height                                int
	invWidth, invHeight, fov, aspectratio, angle float64
	//angle := math.Tan(math.Pi * 0.5 * fov / 180.)
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
func trace(rayorig, raydir vec3.T, spheres *[]sphere, depth int) vec3.T {
	tnear := math.Inf(1)
	//const Sphere* sphere = NULL;
	var sph *sphere
	for i := 0; i < len(*spheres); i++ {
		inter, t0, t1 := (*spheres)[i].intersect(&rayorig, &raydir)
		if inter {
			if t0 < 0 {
				t0 = t1
			}
			if t0 < tnear {
				tnear = t0
				sph = &(*spheres)[i]
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
	} else {
		for i := 0; i < len(*spheres); i++ {
			if (*spheres)[i].emissionsColor[0] > 0 {
				transmission := vec3.T{1, 1, 1}
				lightDirection := vec3.Sub(&(*spheres)[i].centre, &phit)
				lightDirection.Normalize()
				for j := 0; j < len((*spheres)); j++ {
					if i != j {
						org := vec3.Add(&phit, mulF(nhit, bias))
						ints, _, _ := (*spheres)[j].intersect(&org, &lightDirection)
						if ints {
							transmission = vec3.Zero
							break
						}
					}
				}
				ste := vec3.Mul(&sph.surfaceColor, &(*spheres)[i].emissionsColor)
				surfaceColor = vec3.Add(&surfaceColor, mulF(vec3.Mul(&ste, &transmission), math.Max(0.0, float64(vec3.Dot(&nhit, &lightDirection)))))
			}
		}
	}

	return vec3.Add(&surfaceColor, &sph.emissionsColor)
}
func saveImg(imgc chan *image.RGBA, iterc chan int) {
	for {
		img, more := <-imgc
		if more {
			iteration := <-iterc
			f, err := os.Create(fmt.Sprintf("pics/draw%v.jpeg", iteration))
			if err != nil {
				panic(err)
			}
			jpeg.Encode(f, img, nil)
			f.Close()
			fmt.Printf("saved: %v\n", iteration)
		} else {
			return
		}
	}
}
func makeImg(pixel, x, y int, imageg []vec3.T, img *image.RGBA) {
	img.Set(x, y,
		color.RGBA{uint8(math.Min(float64(1), float64(imageg[pixel][0])) * 255),
			uint8(math.Min(float64(1), float64(imageg[pixel][1])) * 255),
			uint8(math.Min(float64(1), float64(imageg[pixel][2])) * 255), 255})
}
func render(camra *cam, spheres *[]sphere, iteration int, imgc chan *image.RGBA, iterc chan int) {
	defer wg.Done()
	img := image.NewRGBA(image.Rect(0, 0, camra.width, camra.height))
	imageg := make([]vec3.T, camra.width*camra.height)
	pixel := 0
	var xx, yy float64
	for y := 0; y < camra.height; y++ {
		for x := 0; x < camra.width; x++ {
			xx = (2*((float64(x)+0.5)*camra.invWidth) - 1) * camra.angle * camra.aspectratio
			yy = (1 - 2*((float64(y)+0.5)*camra.invHeight)) * camra.angle
			rayDir := vec3.T{float32(xx), float32(yy), -1}
			rayDir.Normalize()
			imageg[pixel] = trace(vec3.Zero, rayDir, spheres, 0)
			makeImg(pixel, x, y, imageg, img)
			pixel++
		}
	}
	fmt.Printf("Finished Rendering: %v\n", iteration)
	imgc <- img
	iterc <- iteration
	//go saveImg(img, iteration)

}
func start() {
	interations := 1000.0
	imgc := make(chan *image.RGBA)
	iterc := make(chan int)
	//	width, height                                int
	//invWidth, invHeight, fov, aspectratio, angle float64
	width, height, fov := 1920, 1080, 30.0
	camra := cam{width, height, 1 / float64(width), 1 / float64(height), 30, float64(width) / float64(height), math.Tan(math.Pi * 0.5 * fov / 180.)}
	go saveImg(imgc, iterc)
	for i := 0; i < int(interations); i++ {
		fmt.Printf("Started Rendering: %v\n", i)
		var spheres []sphere

		spheres = append(spheres, makeSphereEmis(vec3.T{0.0, -10004, -10}, 10000, vec3.T{0.0, 0.20, 0.}, vec3.T{0.0, 0.20, 0.0}, 1, 0))
		spheres = append(spheres, makeSphere(vec3.T{0.0, 4.0 - 5, -10}, 1, vec3.T{float32(float64(i) / interations), 0.32, 0.36}, 1, 0.5))
		spheres = append(spheres, makeSphere(vec3.T{5.0, -1, -5}, 2, vec3.T{0.9, 0.76, 0.46}, 1, 0))
		spheres = append(spheres, makeSphere(vec3.T{5.0, 0, -15}, 3, vec3.T{0.65, 0.77, 0.97}, 1, 0))
		wg.Add(1)
		go render(&camra, &spheres, i, imgc, iterc)

	}
	wg.Wait()
	close(imgc)
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	start()

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		runtime.GC()    // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}

}
