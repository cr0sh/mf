// +build ignore

package main

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/cr0sh/mf"
)

const help = `
MF-tools v1.0

Command usage:
m2b: convert MF to BF
b2m: convert BF to MF
`

func main() {
	if len(os.Args) < 3 {
		fmt.Println(help)
		return
	}
	cmd := os.Args[1]
	switch cmd {
	case "m2b":
		fp, err := os.Create(os.Args[2][0:len(os.Args[2])-len(path.Ext(os.Args[2]))] + ".b")
		if err != nil {
			fmt.Println("error:", err)
			return
		}
		fpp, err := os.Open(os.Args[2])
		if err != nil {
			fmt.Println("error:", err)
			return
		}
		r := mf.NewBFWriter(fp)
		io.Copy(r, fpp)
		fpp.Close()

	case "b2m":
		fp, err := os.Create(os.Args[2][0:len(os.Args[2])-len(path.Ext(os.Args[2]))] + ".mf")
		if err != nil {
			fmt.Println("error:", err)
			return
		}
		fpp, err := os.Open(os.Args[2])
		if err != nil {
			fmt.Println("error:", err)
			return
		}
		r := mf.NewBFReader(fp, 4096)
		io.Copy(r, fpp)
		r.Close()
		fpp.Close()
	default:
		fmt.Println(help)
	}
}
