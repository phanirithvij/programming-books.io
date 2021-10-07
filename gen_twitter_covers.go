package main

import (
	"image"
	"image/png"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
)

func loadImageMust(path string) image.Image {
	r, err := os.Open(path)
	panicIfErr(err)
	defer r.Close()
	img, _, err := image.Decode(r)
	panicIfErr(err)
	return img
}

func saveImageAsPNGMust(path string, img image.Image) {
	w, err := os.Create(path)
	panicIfErr(err)
	defer func() {
		err = w.Close()
		panicIfErr(err)
	}()
	err = png.Encode(w, img)
	panicIfErr(err)
}

func optiImageMust(path string) {
	cmd := exec.Command("optipng", "-o7", path)
	_, err := cmd.CombinedOutput()
	panicIfErr(err)
}

func saveImageAsPNGAndOptimize(path string, img image.Image) {
	saveImageAsPNGMust(path, img)
	optiImageMust(path)
	logf(ctx(), "Saved and optimized '%s'\n", path)
}

func getSubimage(img image.Image, r image.Rectangle) image.Image {
	switch im := img.(type) {
	case *image.NRGBA:
		return im.SubImage(r)
	case *image.Paletted:
		return im.SubImage(r)
	}
	panicIf(true, "unsupported image type %T", img)
	return img
}

func genTwitterImage(img image.Image) image.Image {
	r := img.Bounds()
	dx := r.Dx()
	r = image.Rectangle{
		Min: image.Point{0, 0},
		Max: image.Point{dx, dx},
	}
	return getSubimage(img, r)
}

func printImageInfo(path string, img image.Image) {
	r := img.Bounds()
	logf(ctx(), "%s: %v %T\n", path, r, img)
}

func getExistingImagesMust(dir string) map[string]bool {
	m := make(map[string]bool)
	fileInfos, err := ioutil.ReadDir(dir)
	panicIfErr(err)
	for _, fi := range fileInfos {
		name := fi.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext == ".png" {
			m[name] = true
		}
	}
	return m
}

// returns a list of cover images (.png files that are not @2x versions)
func getCoversListMust(dir string) []string {
	var res []string
	fileInfos, err := ioutil.ReadDir(dir)
	panicIfErr(err)
	for _, fi := range fileInfos {
		name := fi.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".png" {
			continue
		}
		if strings.Contains(name, "@2x") {
			continue
		}
		res = append(res, name)
	}
	return res
}

func genTwitterImagesAndExit() {
	srcDir := "covers"
	dstDir := filepath.Join(srcDir, "twitter")
	createDirMust(dstDir)
	covers := getCoversListMust(srcDir)
	existingImages := getExistingImagesMust(dstDir)
	// logf(ctx(), "covers: %v\n", covers)
	for _, coverName := range covers {
		dstPath := filepath.Join(dstDir, coverName)
		if _, ok := existingImages[coverName]; false && ok {
			logf(ctx(), "%s already exists as %s\n", coverName, dstPath)
			continue
		}
		path := filepath.Join(srcDir, coverName)
		img := loadImageMust(path)
		printImageInfo(path, img)
		sub := genTwitterImage(img)
		saveImageAsPNGAndOptimize(dstPath, sub)
	}
	os.Exit(0)
}

func resize(src image.Image, dstSize image.Point) *image.RGBA {
	srcRect := src.Bounds()
	dstRect := image.Rectangle{
		Min: image.Point{0, 0},
		Max: dstSize,
	}
	dst := image.NewRGBA(dstRect)
	draw.CatmullRom.Scale(dst, dstRect, src, srcRect, draw.Over, nil)
	return dst
}

func getProportionalY(p image.Point, x int) int {
	res := (int64(p.Y) * int64(x)) / int64(p.X)
	return int(res)
}

func genSmallCoverImage(img image.Image) *image.RGBA {
	size := img.Bounds().Size()
	x := 140
	y := getProportionalY(size, x)
	return resize(img, image.Point{x, y})
}

// this is one-time code to generate 140 x 198 small images from 595 x 842
// source images. Small images are used in
func genSmallCoversAndExit() {
	srcDir := "covers"
	dstDir := filepath.Join(srcDir, "covers_small")
	createDirMust(dstDir)
	covers := getCoversListMust(srcDir)
	existingImages := getExistingImagesMust(dstDir)
	// logf(ctx(), "covers: %v\n", covers)
	for _, coverName := range covers {
		dstPath := filepath.Join(dstDir, coverName)
		if _, ok := existingImages[coverName]; false && ok {
			logf(ctx(), "%s already exists as %s\n", coverName, dstPath)
			continue
		}
		path := filepath.Join(srcDir, coverName)
		img := loadImageMust(path)
		printImageInfo(path, img)
		sub := genSmallCoverImage(img)
		saveImageAsPNGAndOptimize(dstPath, sub)
	}
	os.Exit(0)
}
