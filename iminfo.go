// Package main parses image tree blob files.
package main

import (
	"github.com/kph/fit"
	"fmt"
	"io/ioutil"
	"os"
)

func listImages(imageList []*fit.ImageLoad) {
	for _, image := range imageList {
		fmt.Printf("listImages: %s: Description=%s Type=%s Arch=%s OS=%s Compression=%s LoadAddr=%x\n", image.Image.Name, image.Image.Description, image.Image.Type, image.Image.Arch, image.Image.Os, image.Image.Compression, image.LoadAddr)
	}
}


func main() {
	b, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	fit := fit.Parse(b)

	fmt.Printf("Description = %s\nAddressCells = %d\nTimeStamp = %s\n", fit.Description, fit.AddressCells, fit.TimeStamp)

	for name, cfg := range fit.Configs {
		fmt.Printf("Configuration %s:%s\n", name, (*cfg).Description)
		listImages(cfg.ImageList)
	}

	fmt.Printf("Hello Universe!\n")
}
