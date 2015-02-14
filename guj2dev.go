package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
)

func cvt(in, out *os.File) {
	guj2dev := 0xa80 - 0x900
	i := bufio.NewReader(in)
	o := bufio.NewWriter(out)
	for {
		r, _, err := i.ReadRune()
		if err != nil {
			o.Flush()
			return
		}
		if rune(0x0a80) <= r && r <= rune(0xAFF) {
			r = rune(int(r) - guj2dev)
		}
		_,err = o.WriteRune(r)
	}
}

func errExit(msg string) {
	fmt.Fprintf(os.Stderr, "%s", msg)
	os.Exit(1)
}


func usage() {
	errExit("Usage: guj2dev input output\n")
}

func main() {
	flag.Parse()
	if flag.NArg() < 2 {
		usage()
	}
	input := flag.Arg(0)
	output := flag.Arg(1)
	in, err := os.Open(input)
	if err != nil {
		errExit(input + ": " + err.Error())
	}
	defer in.Close()
	out, err := os.Create(flag.Arg(1))
	if err != nil {
		errExit(output + ": " + err.Error())
	}
	defer out.Close()
	cvt(in, out)
}
