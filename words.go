package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

var (
	re = regexp.MustCompile("[^ \t]+")
)

type affix struct {
	strip		string
	affix		string
	cond		string
	re		*regexp.Regexp
	morph		[]string
}
type rule struct {
	name		string
	cross		bool	
	ln		int
	suffix		bool
	affix		[]*affix
}

var rules map[string]*rule

var flagv = flag.Bool("v", false, "print verbose output")
var verbose bool

type affixReader struct {
	r	*bufio.Reader
	ln	int
}

func (a *affix)String() string {
	s := fmt.Sprintf("  strip:%v affix:%v cond:%v", a.strip, a.affix, a.cond)
	if a.morph != nil {
		return s + " morph:" + fmt.Sprintf("%v", a.morph)
	}
	return s;
}

func (r *rule)String() string {
	s := fmt.Sprintf("name:%v cross:%v ln:%v suffix:\n", r.name, r.cross, r.ln)
	for _,a := range r.affix {
		s = s + fmt.Sprintf("%v\n", a)
	}
	return s
}

func (a *affixReader) getRule() []string {
	for {
		line, _, err := a.r.ReadLine()
		if err == io.EOF {
			return nil
		}
		a.ln++
		arg := re.FindAllString(string(line), -1)
		if arg == nil || len(arg) == 0 {
			continue
		}
		if arg[0] == "PFX" || arg[0] == "SFX" {
			return arg
		}
	}
}


func processRules(f *os.File) {
	a := &affixReader{bufio.NewReader(f), 0}
	rules = make(map[string]*rule)
	for {
		arg := a.getRule()
		if arg == nil {
			break
		}
		if verbose {
			fmt.Printf("%v\n", arg)
		}
		flag := arg[1]
		r, ok := rules[flag]
		if ok {
			fmt.Printf("%s: duplicate rule @ line %d\n", flag, r.ln)
		}
		count, err := strconv.Atoi(arg[3])
		if err != nil {
			fmt.Printf("%s: badcount: %v\n", flag, arg[3])
			continue
		}
		r = &rule{name:flag,ln:a.ln, affix:make([]*affix, count)}
		if arg[2] == "Y" {
			r.cross = true
		}
		if arg[0] == "SFX" {
			r.suffix = true
		}
		for i := range r.affix {
			arg := a.getRule()
			if arg == nil {
				fmt.Printf("%s: file too short\n", flag)
				panic("bad affix file")
			}
			x := &affix{strip:arg[2], affix:arg[3]}
			if arg[2] == "0" {
				arg[2] = ""
			}
			x = &affix{strip:arg[2], affix:arg[3]}
			if arg[4] != "." {
				x.cond = arg[4]
				if r.suffix {
					x.re, err = regexp.Compile(".*"+x.cond+"$")
				} else {
					x.re, err = regexp.Compile("^"+x.cond+".*")
				}
				if err != nil {
					fmt.Printf("%s:%d: %v\n", err)
				}
			}
			r.affix[i] = x
		}
		rules[flag] = r
	}
	if verbose {
		fmt.Printf("rules:\n")
		for _, r := range rules {
			fmt.Printf("%v\n", r)
		}
	}
}

func processDict(f *os.File) error {
	d := bufio.NewReader(f)
	line, _, err := d.ReadLine()
	if err != nil {
		return err
	}
	count, err := strconv.Atoi(string(line))
	if err != nil {
		panic(fmt.Sprintf("error: %v", err.Error()))
	}
	if verbose {
		fmt.Printf("%d words\n", count)
	}
	words := make([]string, count)
	affix := make([]string, count)
	for i := 0; ; i++ {
		line, _, err = d.ReadLine()
		if err  == io.EOF {
			break
		}
		s := strings.Split(string(line), "/")
		words[i] = s[0]
		if len(s) > 1 {
			affix[i] = s[1]
		}
		if verbose {
			fmt.Printf("%d:%s/%s\n",i,words[i],affix[i])
		}
		fmt.Printf("%s\n", words[i])
		if len(s) > 1 {
			for _,x := range s[1] {
				key := fmt.Sprintf("%c", x)
				//fmt.Printf("key: %v\n", key)
				r, ok := rules[key]
				if !ok {
					fmt.Printf("%s not found\n", key)
					continue
				}
				//fmt.Printf("%v\n", r)
				for _,a := range r.affix {
					x := s[0]
					if r.suffix {
						if a.cond == "" {
							if a.strip != "" {
								x = x[:len(x)-len(a.strip)]
							}
							fmt.Printf("%s%s\n", x, a.affix)
							break
						}

						if a.re.FindString(s[0]) == "" {
							continue
						}
						if a.strip != "" {
							x = x[:len(x)-len(a.strip)]
						}
						fmt.Printf("%s%s\n", x, a.affix)
					} else {
						if a.cond == "" {
							if a.strip != "" {
								x = x[len(a.strip):]
							}
							fmt.Printf("%s%s\n", a.affix,x)
							break
						}
						if a.re.FindString(s[0]) == "" {
							continue
						}
						if a.strip != "" {
							x = x[len(a.strip):]
						}
						fmt.Printf("%s%s\n", a.affix,x)
					}
				}
			}
		}
	}
	return nil
}

func usage() {
	panic("Usage: words aff_file dic_file\n")
}

func main() {
	flag.Parse()
	verbose = *flagv
	if flag.NArg() < 2 {
		usage()
	}
	aff := flag.Arg(0)
	dic := flag.Arg(1)
	if path.Ext(aff) != ".aff" || path.Ext(dic) != ".dic" {
		usage()
	}
	xaff, err := os.Open(aff)
	if err != nil {
		panic(aff + ":" + err.Error())
	}
	xdict, err := os.Open(dic)
	if err != nil {
		panic(dic + ":" + err.Error())
	}
	processRules(xaff)
	processDict(xdict)
	xaff.Close()
	xdict.Close()
	
}
