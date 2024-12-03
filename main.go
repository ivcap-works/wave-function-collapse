package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/zfedoran/go-wfc/pkg/wfc"
)

const TileSetDir = "./tiles"

var VERSION = os.Getenv("VERSION")

var constraintFn = wfc.GetConstraintFunc(3)

func collapseWave(
	images []image.Image,
	width int,
	height int,
	attempts int,
) (image.Image, error) {
	// Setup the initialized state
	wave := wfc.NewWithCustomConstraints(images, width, height, constraintFn)
	initializeWave(wave, "43c0ab1c")

	// Collapse the wave function (make up to 100 attempts)
	err := wave.Collapse(attempts)
	// don't worry about error, we create the image with embedded error info as well
	output := wave.ExportImage()
	return output, err
}

func initializeWave(w *wfc.Wave, borderHex string) {
	w.PossibilitySpace = make([]*wfc.Slot, w.Width*w.Height)
	border := wfc.GetConstraintFromHex(borderHex)

	filterF := func(x int, y int, m *wfc.Module) *wfc.Module {
		if y == 0 && m.Adjacencies[wfc.Up] != border {
			return nil
		}
		if y == w.Height-1 && m.Adjacencies[wfc.Down] != border {
			return nil
		}
		if x == 0 && m.Adjacencies[wfc.Left] != border {
			return nil
		}
		if x == w.Width-1 && m.Adjacencies[wfc.Right] != border {
			return nil
		}
		return m
	}

	for x := 0; x < w.Width; x++ {
		for y := 0; y < w.Height; y++ {
			modules := make([]*wfc.Module, 0)
			for _, m := range w.Input {
				m = filterF(x, y, m)
				if m != nil {
					modules = append(modules, m)
				}
			}
			slot := wfc.Slot{
				X: x, Y: y,
				Superposition: modules,
			}
			w.PossibilitySpace[x+y*w.Width] = &slot
		}
	}
}

func printAdjacencyHashValues() {
	fmt.Printf("Adjacency hash values:\n\n")

	images, err := wfc.LoadImageFolder(TileSetDir)
	if err != nil {
		panic(err)
	}
	files, err := os.ReadDir(TileSetDir)
	if err != nil {
		panic(err)
	}

	// We could use pretty table to do this, but this is just a demo and I don't
	// want the extra dependency.

	fmt.Println("|---------------|----------|----------|")
	fmt.Println("|Tile           |Direction |Hash      |")
	fmt.Println("|---------------|----------|----------|")
	for i, img := range images {
		fn := files[i].Name()
		for _, d := range wfc.Directions {
			c := constraintFn(img, d)
			b := img.Bounds().Max
			fmt.Printf("|%s\t|%s\t   | %s | %dx%d\n", fn, d.ToString(), c, b.X, b.Y)
		}
		fmt.Printf("|- - - - - - - -|- - - - - |- - - - - |\n")
	}
	fmt.Printf("|---------------|----------|----------|\n\n")
}

func load_tiles() ([]image.Image, error) {
	return wfc.LoadImageFolder(TileSetDir)
}

func run_local() {
	output_image := "./output/%d.png"

	// Generate an image from the tileset.
	images, err := load_tiles()
	if err != nil {
		panic(err)
	}

	width := 8
	height := 8
	attempts := 400
	output, err := collapseWave(images, width, height, attempts)
	if err != nil {
		// don't exit if there is an image as well
		fmt.Printf("ERROR:\tUnable to generate: %v", err)
	}
	seed := int(time.Now().UnixNano())
	output_file := fmt.Sprintf(output_image, seed)
	if err = wfc.SaveImage(output_file, output); err != nil {
		fmt.Printf("ERROR:\tUnable to save image: %v\n", err)
		os.Exit(-1)
	}
	fmt.Printf("INFO:\tImage saved to: %s\n", output_file)
}

type Request struct {
	Width    int `json:"width,omitempty"`
	Height   int `json:"height,omitempty"`
	Attempts int `json:"attempts,omitempty"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		req := parse_request(w, r)
		if req == nil {
			return
		}
		images, err := load_tiles()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		output, err := collapseWave(images, req.Width, req.Height, req.Attempts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		save_image(output, w)
		return
	} else {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func parse_request(w http.ResponseWriter, r *http.Request) *Request {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}
	var request Request
	if err = json.Unmarshal(body, &request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil
	}
	if request.Height == 0 {
		request.Height = 8
	}
	if request.Width == 0 {
		request.Width = 8
	}
	if request.Attempts == 0 {
		request.Attempts = 400
	}
	return &request
}

func save_image(img image.Image, w http.ResponseWriter) {
	buf := new(bytes.Buffer)
	err := png.Encode(buf, img)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	_, err = buf.WriteTo(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	var runOnce = flag.Bool("run-once", false, "Creaste map and exit")
	var printAdjacency = flag.Bool("print-adjacency", false, "Print adjacency for the tileset and exit")
	var port = flag.Int("port", 8080, "Port to listen on")

	flag.Parse()

	if *runOnce {
		run_local()
		os.Exit(0)
	}
	if *printAdjacency {
		printAdjacencyHashValues()
	}

	fmt.Printf("INFO:\tListening on port %d\n", *port)
	http.HandleFunc("/", handler)
	http.HandleFunc("/_healtz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		s := fmt.Sprintf("{\"version\": \"%s\"}\n", VERSION)
		w.Write([]byte(s))
	})
	if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil); err != nil {
		fmt.Printf("ERROR:\tServer problem - %v'\n", err)
	}
}
