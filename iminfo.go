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
)

type Fit struct {
	fdt	*fdt.Tree
}

type Image struct {
	Name	string
	Type	string
	Arch	string
}

func debugDumpProperties(n *fdt.Node) {
	for name, value := range n.Properties {
		fmt.Printf("[DEBUG]%s: %s = %q\n", n.Name, name, value)
	}
}

func debugDumpNode(n *fdt.Node) {

	debugDumpProperties(n)

	for _, c := range n.Children {
		if strings.HasPrefix(c.Name, "conf") {
			debugDumpProperties(c)
		}
	}
}

func nodeToString(b []byte) (s string) {
	return strings.Split(string(b), "\x00")[0]
}


// validateHash takes a hash node, and attempts to validate it. It takes
func (f *Fit)validateHash(n *fdt.Node, data []byte) (err error) {
	debugDumpProperties(n)
	
	algo,ok := n.Properties["algo"]
	if !ok {
		return errors.New("algo property missing")
	}

	value,ok := n.Properties["value"]
	if !ok {
		return errors.New("value property missing")
	}

	algostr := f.fdt.PropString(algo)

	fmt.Printf("Checking %s %v... ", algostr, value)
	if algostr == "sha1" {
		shasum := sha1.Sum(data)
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
		calcsum := crc32.ChecksumIEEE(data)
		if calcsum != propsum {
			fmt.Printf("incorrect, expected %d calculated %d", propsum, calcsum)
			return fmt.Errorf("crc32 incorrect, expected %d calculated %d", propsum, calcsum)
		}
		fmt.Printf("OK!\n")
		return
	}

	if algostr == "md5" {
		md5sum := md5.Sum(data)
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

func (f *Fit)validateHashes(n *fdt.Node) (err error) {
	data,ok := n.Properties["data"]
	if !ok {
		return errors.New("data property missing")
	}

	for _, c := range n.Children {
		if c.Name == "hash" || strings.HasPrefix(c.Name, "hash@") {
			err = f.validateHash(c, data)
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
	i.Type = f.fdt.PropString(node.Properties["type"])
	i.Arch = f.fdt.PropString(node.Properties["arch"])

	err := f.validateHashes(node)
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

func listImages(imageList []*Image) {
	for _, image := range imageList {
		fmt.Printf("listImages: %s:%s\n", image.Name, image.Type)
	}
}

// DumpRoot blah blah blah.
func DumpRoot(t *fdt.Tree) {
	debugDumpNode(t.RootNode)
}

func main() {
	b, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	fit := Fit{}
	fit.fdt = &fdt.Tree{Debug: false, IsLittleEndian: false}
	err = fit.fdt.Parse(b)

	if err != nil {
		panic(err)
	}

	DumpRoot(fit.fdt)

	imageList, err := fit.parseConfiguration("")
		
	listImages(imageList)

	fit.fdt.MatchNode("configurations", debugDumpNode)

	fmt.Printf("Hello Universe!\n")
}
