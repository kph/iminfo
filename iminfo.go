package main

import (
	"github.com/platinasystems/fdt"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

func gatherConfiguration(c *fdt.Node) {
	for name, value := range c.Properties {
		fmt.Printf("%s: %s = %q\n", c.Name, name, value)
	}
}

func gatherConfigurations(n *fdt.Node) {
	for _, c := range n.Children {
		if strings.Contains(c.Name, "conf") {
			gatherConfiguration(c)
		}
	}
}

func gatherImage(n *fdt.Node) {
	for name, value := range n.Properties {
		if name != "data" {
			fmt.Printf("%s: %s = %q\n", n.Name, name, value)
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
		t.MatchNode("configurations", gatherConfigurations)
		gatherImages(t, "kernel@1", "fdt@1", "ramdisk@1")
	}

	fmt.Printf("Hello Universe!\n")
}
