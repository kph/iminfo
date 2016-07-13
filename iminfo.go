package main

import (
	"github.com/platinasystems/fdt"
	"fmt"
	"io/ioutil"
	"os"
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

func gatherHashes(n *fdt.Node) {
	for name, value := range n.Properties {
		fmt.Printf("%s: %s = %q\n", n.Name, name, value)
	}
}

func gatherImage(n *fdt.Node) {
	for name, value := range n.Properties {
		if name != "data" {
			fmt.Printf("%s: %s = %q\n", n.Name, name, value)
		}
	}
	for _, c := range n.Children {
		if strings.HasPrefix(c.Name, "hash") {
			gatherHashes(c)
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
