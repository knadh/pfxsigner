package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"io/ioutil"
	"log"

	"github.com/knadh/pfxsigner/internal/processor"
	"github.com/unidoc/unipdf/v3/model"
	"github.com/urfave/cli"
)

func initApp(f cli.ActionFunc) cli.ActionFunc {
	return func(c *cli.Context) error {
		// Read JSON config.
		fName := c.GlobalString("props-file")

		log.Printf("loading signature properties from %s", fName)
		b, err := ioutil.ReadFile(fName)
		if err != nil {
			log.Fatalf("error opening properties file %s: %v", fName, err)
		}

		// Make properties.
		pr, err := parseProps(b)
		if err != nil {
			log.Fatalf("error in JSON properties file: %v", err)
		}

		// Initialize global workers.
		proc = processor.New(pr, logger)

		// Load the PFX.
		if err := proc.LoadPFX(c.GlobalString("pfx-file"), c.GlobalString("pfx-password")); err != nil {
			log.Fatalf("error loading PFX: %v", err)
		}

		return f(c)
	}
}

// parseProps loads signature properties from the given JSON blob.
func parseProps(b []byte) (processor.SignProps, error) {
	var pr processor.SignProps
	if err := json.Unmarshal(b, &pr); err != nil {
		return pr, fmt.Errorf("error parsing JSON properties file: %v", err)
	}

	// Validate colours.
	var err error
	pr.Style.FontColorRGBA, err = parseHexColor(pr.Style.FontColor)
	if err != nil {
		return pr, fmt.Errorf("invalid `fontColor`: %v", err)
	}
	pr.Style.BgColorRGBA, err = parseHexColor(pr.Style.BgColor)
	if err != nil {
		return pr, fmt.Errorf("invalid `bgColor`: %v", err)
	}
	pr.Style.BorderColorRGBA, err = parseHexColor(pr.Style.BorderColor)
	if err != nil {
		return pr, fmt.Errorf("invalid `borderColor`: %v", err)
	}

	return pr, nil
}

// parseHexColor converts a hex colour string to RGB.
// https://stackoverflow.com/questions/54197913/parse-hex-string-to-image-color
func parseHexColor(s string) (model.PdfColorDeviceRGB, error) {
	var (
		c   = color.RGBA{A: 0xff}
		err error
	)

	switch len(s) {
	case 7:
		_, err = fmt.Sscanf(s, "#%02x%02x%02x", &c.R, &c.G, &c.B)
	case 4:
		_, err = fmt.Sscanf(s, "#%1x%1x%1x", &c.R, &c.G, &c.B)
		// Double the hex digits:
		c.R *= 17
		c.G *= 17
		c.B *= 17
	default:
		err = fmt.Errorf("invalid colour hex length, must be 7 or 4")

	}
	return model.PdfColorDeviceRGB{
		float64(c.R),
		float64(c.G),
		float64(c.B),
	}, err
}
