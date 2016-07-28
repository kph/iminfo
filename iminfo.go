// Package main parses image tree blob files.
package main

import (
	"bytes"
	"hash/crc32"
	"errors"
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
	Configs		map[string]*Config
}

type Config struct {
	Description	string
	imageList	[]*Image
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

func debugDumpProperties(n *fdt.Node) {
	for name, value := range n.Properties {
		fmt.Printf("[DEBUG]%s: %s = %q\n", n.Name, name, value)
	}
}

func (f *Fit)getProperty(n *fdt.Node, propName string) ([]byte) {
	if val, ok := n.Properties[propName]; ok {
		return val
	}

	panic(fmt.Errorf("Required property %s missing\n", propName))
}

// validateHash takes a hash node, and attempts to validate it. It takes
func (f *Fit)validateHash(n *fdt.Node, i *Image) (err error) {
	debugDumpProperties(n)
	
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

func (f *Fit)parseImage(n *fdt.Node, imageList *[]*Image, imageName string) {
	node,ok := n.Children[imageName]
	if !ok {
		return
	}

	i := &Image{}
	i.Name = imageName
	i.Description = f.fdt.PropString(f.getProperty(node, "description"))
	i.Type = f.fdt.PropString(f.getProperty(node, "type"))
	i.Arch = f.fdt.PropString(f.getProperty(node, "arch"))
	i.Os = f.fdt.PropString(node.Properties["os"])
	i.Compression = f.fdt.PropString(f.getProperty(node, "compression"))
	i.Data = f.getProperty(node, "data")

	err := f.validateHashes(node, i)
	if err != nil {
		panic(err)
	}
	//i.Load = node.Properties["load"]
	//i.Entry = node.Properties["entry"]
	//i.Len = len(node.Properties["data"])

	*imageList = append(*imageList, i)
}

func (f *Fit) parseConfiguration(whichconf string) (imageList []*Image, err error) {
	conf := f.fdt.RootNode.Children["configurations"]
	images := f.fdt.RootNode.Children["images"]

	if (whichconf == "") {
		def, ok := conf.Properties["default"]

		if !ok {
			return nil, errors.New("Can't find default node")
		}

		whichconf = f.fdt.PropString(def)
	}

	fmt.Printf("parseConfiguration %s: %q\n", conf.Name, whichconf)

	conf,ok := conf.Children[whichconf]

	if !ok {
		return nil, fmt.Errorf("Can't find configuration %s", whichconf)
	}

	description := conf.Properties["description"]
	if description != nil {
		fmt.Printf("parseConfiguration %s: %s\n", whichconf, f.fdt.PropString(description))
	}

	imageList = []*Image{}

	kernel := f.fdt.PropString(conf.Properties["kernel"])
	fdt := f.fdt.PropString(conf.Properties["fdt"])
	ramdisk := f.fdt.PropString(conf.Properties["ramdisk"])

	fmt.Printf("parseConfiguration kernel=%s fdt=%s ramdisk=%s\n", kernel, fdt, ramdisk)

	f.parseImage(images, &imageList, kernel)
	f.parseImage(images, &imageList, fdt)
	f.parseImage(images, &imageList, ramdisk)

	return imageList, nil
}

func (f *Fit)parseConfigurations() {
	conf := f.fdt.RootNode.Children["configurations"]

	for _, c := range conf.Children {
		if c.Name == "conf" || strings.HasPrefix(c.Name, "conf@") {
			cfg := Config{}
			imageList, err := f.parseConfiguration(c.Name)
			if err != nil {
				panic(err)
			}
			cfg.imageList = imageList
			f.Configs[c.Name] = &cfg
		}
	}
}

func listImages(imageList []*Image) {
	for _, image := range imageList {
		fmt.Printf("listImages: %s: Description=%s Type=%s Arch=%s OS=%s Compression=%s\n", image.Name, image.Description, image.Type, image.Arch, image.Os, image.Compression)
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

	fit.Configs = make(map[string]*Config)

	fit.parseConfigurations()

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
