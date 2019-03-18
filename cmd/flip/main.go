package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"github.com/flopp/go-findfont"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"html/template"
	"image"
	"image/draw"
	"image/jpeg"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

func main() {
	http.HandleFunc("/", handler)
	log.Println("Listening on 8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

var (
	dpi  = flag.Float64("dpi", 92, "screen resolution in Dots Per Inch")
	size = flag.Float64("size", 72, "font size in points")

	lp = 10
	tp = 6
	bp = 25
	fh = 96
	rp = 10
)

func imageWithLabel(label string) image.Image {
	x := 0
	y := fh

	// Initialize the context.
	fg, bg := image.Black, image.White
	rgba := image.NewRGBA(image.Rect(0, 0, 750, fh))
	draw.Draw(rgba, rgba.Bounds(), bg, image.ZP, draw.Src)
	c := freetype.NewContext()
	c.SetDPI(*dpi)
	c.SetFont(f)
	c.SetFontSize(*size)
	c.SetClip(rgba.Bounds())
	c.SetDst(rgba)
	c.SetSrc(fg)

	pt := freetype.Pt(x, y)

	if n, err := c.DrawString(label, pt); err != nil {
		// handle error
		return rgba
	} else {
		return rgba.SubImage(image.Rect(0, 0, n.X.Round(), n.Y.Round()))
	}
}

func tableFlip(flip, table string) image.Image {

	flipImage := imageWithLabel(flip)
	tableImage := imageWithLabel(table)

	imgX := flipImage.Bounds().Max.X + tableImage.Bounds().Max.X // todo plus padding
	imgY := flipImage.Bounds().Max.Y                             // todo plus padding

	img := image.NewRGBA(image.Rect(0, 0, imgX, imgY))

	oX := 0
	oY := 0
	for x := 0; x < flipImage.Bounds().Max.X; x++ {
		for y := 0; y < flipImage.Bounds().Max.Y; y++ {
			img.Set(x+oX, y+oY, flipImage.At(x, y))
		}
	}
	oX += flipImage.Bounds().Max.X

	maxX := tableImage.Bounds().Max.X
	maxY := tableImage.Bounds().Max.Y
	for x := 0; x < maxX; x++ {
		for y := 0; y < maxY; y++ {
			img.Set(oX+(maxX-1-x), oY+(maxY-1-y), tableImage.At(x, y))
		}
	}

	return img
}

func getQueryParam(r *http.Request, key string) string {
	keys, ok := r.URL.Query()[key]
	if !ok || len(keys[0]) < 1 {
		return ""
	}
	return keys[0]
}

var f *truetype.Font

func init() {
	fontPath, err := findfont.Find("Arial Unicode.ttf")
	if err != nil {
		fontPath = "/var/run/ko/Arial Unicode.ttf"
	}
	log.Print("font: " + fontPath)

	// load the font with the freetype library
	fontData, err := ioutil.ReadFile(fontPath)
	if err != nil {
		panic(err)
	}
	f, err = truetype.Parse(fontData)
	if err != nil {
		panic(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {

	page := getQueryParam(r, "page")

	flip := getQueryParam(r, "flip")
	if flip == "" {
		flip = "┬─┬"
	}

	img := tableFlip("(╯°□°)╯︵ ", flip)

	if page == "html" {
		writeImageWithTemplate(w, &img)
	} else {
		writeImage(w, &img)
	}
}

var ImageTemplate string = `<!DOCTYPE html>
<html lang="en"><head></head>
<body><img src="data:image/jpg;base64,{{.Image}}"></body>`

// Writeimagewithtemplate encodes an image 'img' in jpeg format and writes it into ResponseWriter using a template.
func writeImageWithTemplate(w http.ResponseWriter, img *image.Image) {

	buffer := new(bytes.Buffer)
	if err := jpeg.Encode(buffer, *img, nil); err != nil {
		log.Fatalln("unable to encode image.")
	}

	str := base64.StdEncoding.EncodeToString(buffer.Bytes())
	if tmpl, err := template.New("image").Parse(ImageTemplate); err != nil {
		log.Println("unable to parse image template.")
	} else {
		data := map[string]interface{}{"Image": str}
		if err = tmpl.Execute(w, data); err != nil {
			log.Println("unable to execute template.")
		}
	}
}

// writeImage encodes an image 'img' in jpeg format and writes it into ResponseWriter.
func writeImage(w http.ResponseWriter, img *image.Image) {

	buffer := new(bytes.Buffer)
	if err := jpeg.Encode(buffer, *img, nil); err != nil {
		log.Println("unable to encode image.")
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Length", strconv.Itoa(len(buffer.Bytes())))
	if _, err := w.Write(buffer.Bytes()); err != nil {
		log.Println("unable to write image.")
	}
}
