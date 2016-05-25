// +build ignore

package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"strconv"

	"github.com/cr0sh/mf"
)

const help = `
MF-tools v1.1

Command usage:
m2b <filename> : convert MF to BF
b2m <filename> <memsize> : convert BF to MF
`

const defaultMemsize uint32 = 4096

func main() {
	if len(os.Args) < 3 {
		fmt.Println(help)
		return
	}
	go func() {
		for {
			var buf [4096]byte
			fmt.Scanln()
			runtime.Stack(buf[:], true)
			fmt.Println(string(buf[:]))
		}
	}()

	cmd := os.Args[1]
	switch cmd {
	case "m2b":
		fp, err := os.Create(os.Args[2][0:len(os.Args[2])-len(path.Ext(os.Args[2]))] + "_compile.bf")
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
		if _, err := io.Copy(r, fpp); err != nil {
			fmt.Println("error:", err)
		}
		fpp.Close()

	case "b2m":
		var memsize uint32
		if len(os.Args) < 4 {
			memsize = defaultMemsize
			fmt.Println("warning: setting memsize to default", defaultMemsize)
		} else {
			n, err := strconv.Atoi(os.Args[3])
			if err != nil || n == 0 || uint64(n) >= (uint64(1)<<32) {
				fmt.Println("invalid memsize")
				return
			}
			memsize = uint32(n)
		}
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
		r := mf.NewBFReader(fp, memsize)
		io.Copy(r, fpp)
		r.Close()
		fpp.Close()
	default:
		fmt.Println(help)
	}
}
