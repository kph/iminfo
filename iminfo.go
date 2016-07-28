// Package main parses image tree blob files.
package main

import (
	"bytes"
	"hash/crc32"
	"github.com/platinasystems/fdt"
	"fmt"
	"io/ioutil"
	"crypto/md5"
	"os"
	"crypto/sha1"
	"strings"
	"time"
)

type Fit struct {
	fdt		*fdt.Tree
	Description	string
	AddressCells	uint32
	TimeStamp	time.Time
	DefaultConfig	string
	Images		map[string]*Image
	Configs		map[string]*Config
}

type Config struct {
	Description	string
	imageList	[]*ImageLoad
}

type Image struct {
	Name		string
	Description	string
	Type		string
	Arch		string
	Os		string
	Compression	string
	Data		[]byte
}

type ImageLoad struct {
	Image		*Image
}

func (f *Fit)getProperty(n *fdt.Node, propName string) ([]byte) {
	if val, ok := n.Properties[propName]; ok {
		return val
	}

	panic(fmt.Errorf("Required property %s missing\n", propName))
}

// validateHash takes a hash node, and attempts to validate it. It takes
func (f *Fit)validateHash(n *fdt.Node, i *Image) (err error) {
	algo := f.getProperty(n, "algo")
	value := f.getProperty(n, "value")
	algostr := f.fdt.PropString(algo)

	fmt.Printf("Checking %s %v... ", algostr, value)
	if algostr == "sha1" {
		shasum := sha1.Sum(i.Data)
		shaslice := shasum[:]
		if !bytes.Equal(value, shaslice) {
			fmt.Printf("error, calculated %v!\n", shaslice)
			return fmt.Errorf("sha1 incorrect, expected %v! calculated %v!\n", value, shaslice)
		}
		fmt.Print("OK!\n")
		return
	}

	if algostr == "crc32" {
		propsum := f.fdt.PropUint32(value)
		calcsum := crc32.ChecksumIEEE(i.Data)
		if calcsum != propsum {
			fmt.Printf("incorrect, expected %d calculated %d", propsum, calcsum)
			return fmt.Errorf("crc32 incorrect, expected %d calculated %d", propsum, calcsum)
		}
		fmt.Printf("OK!\n")
		return
	}

	if algostr == "md5" {
		md5sum := md5.Sum(i.Data)
		md5slice := md5sum[:]
		if !bytes.Equal(value, md5slice) {
			fmt.Printf("error, calculated %v!\n", md5slice)
			return fmt.Errorf("sha1 incorrect, expected %v! calculated %v!\n", value, md5slice)
		}
		fmt.Print("OK!\n")
		return
	}

	fmt.Printf("Unknown algorithm!\n")
	return
}

func (f *Fit)validateHashes(n *fdt.Node,i *Image) (err error) {
	for _, c := range n.Children {
		if c.Name == "hash" || strings.HasPrefix(c.Name, "hash@") {
			err = f.validateHash(c, i)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *Fit)parseImage(imageList *[]*ImageLoad, imageName string) {
	il := &ImageLoad{}

	il.Image = f.Images[imageName]

	*imageList = append(*imageList, il)
}

func (f *Fit) parseConfiguration(whichconf string) (imageList []*ImageLoad, err error) {
	conf := f.fdt.RootNode.Children["configurations"]

	fmt.Printf("parseConfiguration %s: %q\n", conf.Name, whichconf)

	conf,ok := conf.Children[whichconf]

	if !ok {
		return nil, fmt.Errorf("Can't find configuration %s", whichconf)
	}

	description := conf.Properties["description"]
	if description != nil {
		fmt.Printf("parseConfiguration %s: %s\n", whichconf, f.fdt.PropString(description))
	}

	imageList = []*ImageLoad{}

	kernel := f.fdt.PropString(conf.Properties["kernel"])
	fdt := f.fdt.PropString(conf.Properties["fdt"])
	ramdisk := f.fdt.PropString(conf.Properties["ramdisk"])

	fmt.Printf("parseConfiguration kernel=%s fdt=%s ramdisk=%s\n", kernel, fdt, ramdisk)

	f.parseImage(&imageList, kernel)
	f.parseImage(&imageList, fdt)
	f.parseImage(&imageList, ramdisk)

	return imageList, nil
}

func listImages(imageList []*ImageLoad) {
	for _, image := range imageList {
		fmt.Printf("listImages: %s: Description=%s Type=%s Arch=%s OS=%s Compression=%s\n", image.Image.Name, image.Image.Description, image.Image.Type, image.Image.Arch, image.Image.Os, image.Image.Compression)
	}
}

func Parse(b []byte) (f *Fit) {
	fit := Fit{}
	fit.fdt = &fdt.Tree{Debug: false, IsLittleEndian: false}
	err := fit.fdt.Parse(b)
	if err != nil {
		panic(err)
	}

	fit.Description = fit.fdt.PropString(fit.getProperty(fit.fdt.RootNode, "description"))
	fit.AddressCells = fit.fdt.PropUint32(fit.getProperty(fit.fdt.RootNode, "#address-cells"))
	fit.TimeStamp = time.Unix(int64(fit.fdt.PropUint32(fit.getProperty(fit.fdt.RootNode, "timestamp"))), 0)

	images := fit.fdt.RootNode.Children["images"]
	fit.Images = make(map[string]*Image)

	for _, image := range images.Children {
		i := Image{}
		i.Name = image.Name
		i.Description = fit.fdt.PropString(fit.getProperty(image, "description"))
		i.Type = fit.fdt.PropString(fit.getProperty(image, "type"))
		i.Arch = fit.fdt.PropString(fit.getProperty(image, "arch"))
		i.Os = fit.fdt.PropString(image.Properties["os"])
		i.Compression = fit.fdt.PropString(fit.getProperty(image, "compression"))
		i.Data = fit.getProperty(image, "data")

		err := fit.validateHashes(image, &i)
		if err != nil {
			panic(err)
		}
		//i.Load = image.Properties["load"]
		//i.Entry = image.Properties["entry"]
		//i.Len = len(image.Properties["data"])
		fit.Images[image.Name] = &i;
	}
	
	conf := fit.fdt.RootNode.Children["configurations"]
	fit.Configs = make(map[string]*Config)

	fit.DefaultConfig = fit.fdt.PropString(fit.getProperty(conf, "default"))

	for _, c := range conf.Children {
		if c.Name == "conf" || strings.HasPrefix(c.Name, "conf@") {
			cfg := Config{}
			imageList, err := fit.parseConfiguration(c.Name)
			if err != nil {
				panic(err)
			}
			cfg.imageList = imageList
			fit.Configs[c.Name] = &cfg
		}
	}

	return &fit
}

func main() {
	b, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	fit := Parse(b)

	fmt.Printf("Description = %s\nAddressCells = %d\nTimeStamp = %s\n", fit.Description, fit.AddressCells, fit.TimeStamp)

	for name, cfg := range fit.Configs {
		fmt.Printf("Configuration %s:%s\n", name, (*cfg).Description)
		listImages(cfg.imageList)
	}

	fmt.Printf("Hello Universe!\n")
}
