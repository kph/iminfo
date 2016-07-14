package main

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"github.com/platinasystems/fdt"
	"fmt"
	"io/ioutil"
	"os"
	"crypto/sha1"
	"strings"
)

func gatherConfiguration(n *fdt.Node) {
	for name, value := range n.Properties {
		fmt.Printf("%s: %s = %q\n", n.Name, name, value)
	}
}

func gatherConfigurations(n *fdt.Node) {
	for name, value := range n.Properties {
		fmt.Printf("%s: %s = %q\n", n.Name, name, value)
	}

	for _, c := range n.Children {
		if strings.HasPrefix(c.Name, "conf") {
			gatherConfiguration(c)
		}
	}
}

func gatherHashes(n *fdt.Node, data []byte) {
	for name, value := range n.Properties {
		fmt.Printf("%s: %s = %q\n", n.Name, name, value)
	}

	algo,ok := n.Properties["algo"]
	if !ok {
		panic("algo property missing")
	}

	value,ok := n.Properties["value"]
	if !ok {
		panic("value property missing")
	}

	algostr := string(algo[0:len(algo)-1])

	if algostr == "sha1" {
		fmt.Print("Algo is sha1\n")
		shasum := sha1.Sum(data)
		shaslice := shasum[:]
		if bytes.Equal(value, shaslice) {
			fmt.Printf("sha1 correct: %v!\n", value)
		}
	}

	if algostr == "crc32" {
		fmt.Print("Algo is crc32\n")
		calcsum := crc32.ChecksumIEEE(data)
		propsum := binary.BigEndian.Uint32(value)
		if calcsum == propsum {
			fmt.Printf("crc32 correct: %d!\n", propsum);
		}
	}
}

func gatherImage(n *fdt.Node) {
	for name, value := range n.Properties {
		if name != "data" {
			fmt.Printf("%s: %s = %q\n", n.Name, name, value)
		}
	}
	data,ok := n.Properties["data"]
	if !ok {
		panic("Can't find data property")
	}
	for _, c := range n.Children {
		if strings.HasPrefix(c.Name, "hash") {
			gatherHashes(c, data)
		}
	}
}

func gatherImages(t *fdt.Tree, kernel string, fdt string, ramdisk string) {
	t.MatchNode(kernel, gatherImage)
	t.MatchNode(fdt, gatherImage)
	if ramdisk != "" {
		t.MatchNode(ramdisk, gatherImage)
	}
}

func parseConfiguration(n *fdt.Node) {
	def, ok := n.Properties["default"]
	if !ok {
		panic("Can't find default node")
	}
	fmt.Printf("parseConfiguration %s: %q\n", n.Name, def)

	defstr := string(def[0:len(def)-1])
	
	conf,ok := n.Children[defstr]

	if !ok {
		panic("Can't find default configuration")
	}

	gatherConfiguration(conf)
}

func main() {
	b, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	t := &fdt.Tree{Debug: false, IsLittleEndian: false}
	t.Parse(b)

	if false {
		fmt.Printf("%v\n", t)
		panic(t)
	}

	if true {
		for name, value := range t.RootNode.Properties {
			fmt.Printf("foo %s: %s = %q\n", t.RootNode.Name, name, value)
		}

		for _, c := range t.RootNode.Children {
			fmt.Printf("bar %s\n", c.Name)
		}

		parseConfiguration(t.RootNode.Children["configurations"])
		
		t.MatchNode("configurations", gatherConfigurations)
		gatherImages(t, "kernel@1", "fdt@1", "ramdisk@1")
	}

	fmt.Printf("Hello Universe!\n")
}
