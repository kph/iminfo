// Package main parses image tree blob files.
package main

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"errors"
	"github.com/platinasystems/fdt"
	"fmt"
	"io/ioutil"
	"os"
	"crypto/sha1"
	"strings"
)

func debugDumpProperties(n *fdt.Node) {
	for name, value := range n.Properties {
		fmt.Printf("%s: %s = %q\n", n.Name, name, value)
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

// validateHashes takes a hash node, and attempts to validate it. It takes
func validateHashes(n *fdt.Node, data []byte) (err error) {
	debugDumpProperties(n)
	
	algo,ok := n.Properties["algo"]
	if !ok {
		return errors.New("algo property missing")
	}

	value,ok := n.Properties["value"]
	if !ok {
		return errors.New("value property missing")
	}

	algostr := string(algo[0:len(algo)-1])

	if algostr == "sha1" {
		shasum := sha1.Sum(data)
		shaslice := shasum[:]
		fmt.Printf("Checking sha1 %v... ", value)
		if !bytes.Equal(value, shaslice) {
			fmt.Printf("error, calculated %v!\n", shaslice)
			return fmt.Errorf("sha1 incorrect, expected %v! calculated %v!\n", value, shaslice)
		}
		fmt.Print("OK!\n")
		return
	}

	if algostr == "crc32" {
		propsum := binary.BigEndian.Uint32(value)
		fmt.Printf("Checking crc32 %d... ", propsum)
		calcsum := crc32.ChecksumIEEE(data)
		if calcsum != propsum {
			fmt.Printf("incorrect, expected %d calculated %d", propsum, calcsum)
			return fmt.Errorf("crc32 incorrect, expected %d calculated %d", propsum, calcsum)
		}
		fmt.Printf("OK!\n")
		return
	}

	return
}

func gatherImage(n *fdt.Node) (err error) {
	for name, value := range n.Properties {
		if name != "data" {
			fmt.Printf("%s: %s = %q\n", n.Name, name, value)
		}
	}
	data,ok := n.Properties["data"]
	if !ok {
		return errors.New("data property missing")
	}

	for _, c := range n.Children {
		if strings.HasPrefix(c.Name, "hash") {
			err = validateHashes(c, data)
			if err != nil {
				return err
			}
		}
	}
	return
}

func gatherImageHelper(n *fdt.Node) {
	gatherImage(n)
}

func gatherImages(t *fdt.Tree, kernel string, fdt string, ramdisk string) {
	t.MatchNode(kernel, gatherImageHelper)
	t.MatchNode(fdt, gatherImageHelper)
	if ramdisk != "" {
		t.MatchNode(ramdisk, gatherImageHelper)
	}
}

func parseConfiguration(n *fdt.Node) (error) {
	def, ok := n.Properties["default"]
	if !ok {
		return errors.New("Can't find default node")
	}
	fmt.Printf("parseConfiguration %s: %q\n", n.Name, def)

	defstr := string(def[0:len(def)-1])
	
	conf,ok := n.Children[defstr]

	if !ok {
		return errors.New("Can't find default configuration")
	}

	debugDumpNode(conf)

	return nil
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

	t := &fdt.Tree{Debug: false, IsLittleEndian: false}
	t.Parse(b)

	if false {
		fmt.Printf("%v\n", t)
		panic(t)
	}

	if true {
		DumpRoot(t)
		parseConfiguration(t.RootNode.Children["configurations"])
		
		t.MatchNode("configurations", debugDumpNode)
		gatherImages(t, "kernel@1", "fdt@1", "ramdisk@1")
	}

	fmt.Printf("Hello Universe!\n")
}
