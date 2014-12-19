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
	spaceRE = regexp.MustCompile("[^ \t]+")
	commaRE = regexp.MustCompile("[^ \t,]+")
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
	need		bool
	affix		[]*affix
}

var rules map[string]*rule

var flagv = flag.Bool("v", false, "print verbose output")
var verbose bool

var (
	numflags bool
	flaglen = 1
	needaffix string	// tags stems in a dictionary
)

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
		arg := spaceRE.FindAllString(string(line), -1)
		if arg == nil || len(arg) == 0 || arg[0][0] == '#' {
			continue
		}
		//fmt.Fprintf(os.Stderr, "%d '%s'\n", len(arg), arg[0])
		switch arg[0] {
		default:
			fmt.Fprintf(os.Stderr, "%d: invalid option: %s\n", a.ln, arg[0])
		case "FLAG":
			if len(arg) == 1 {
				fmt.Fprintf(os.Stderr, "%d: FLAG needs an argument\n", a.ln)
				continue
			}
			switch arg[1] {
			case "num":
				numflags = true
			case "long":
				flaglen = 2
			default:
				fmt.Fprintf(os.Stderr, "%d: invalid FLAG value: %s\n", a.ln, arg[1])
			}

		case "PFX", "SFX":
			return arg

		case "SET":
			if len(arg) == 1 || arg[1] != "UTF-8" {
				fmt.Fprintf(os.Stderr, "%d: only UTF-8 can be set\n", a.ln)
			}

		case "TRY": // ignore for now
		case "NEEDAFFIX":
			if len(arg) < 2 {
				fmt.Fprintf(os.Stderr, "%d: needs a flag arg\n", a.ln)
				continue
			}
			if needaffix != "" {
				fmt.Fprintf(os.Stderr, "%d: NEEDAFIX already specified as %s\n", a.ln, needaffix)
				continue
			}
			needaffix = arg[1]
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
			fmt.Fprintf(os.Stderr, "%v\n", arg)
		}
		flag := arg[1]
		r, ok := rules[flag]
		if ok {
			fmt.Fprintf(os.Stderr, "%s: duplicate rule @ line %d\n", flag, r.ln)
		}
		count, err := strconv.Atoi(arg[3])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: badcount: %v\n", flag, arg[3])
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
				fmt.Fprintf(os.Stderr, "%s: file too short\n", flag)
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
					fmt.Fprintf(os.Stderr, "%s:%d: %v\n", err)
				}
			}
			r.affix[i] = x
		}
		rules[flag] = r
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "rules:\n")
		for _, r := range rules {
			fmt.Fprintf(os.Stderr, "%v\n", r)
		}
	}
	if needaffix != "" {
		rules[needaffix] = &rule{name:needaffix, need: true}
	}
}

func getflags(aff string) (flags []string) {
	if numflags {
		flags = commaRE.FindAllString(aff, -1)
	} else {
		flags = make([]string, len(aff)/flaglen)
		j := 0
		for i,_ := range flags {
			flags[i] = aff[j:j+flaglen]
			j += flaglen
		}
	}
	return
}

func expand(line string) {
	s := strings.Split(line, "/")
	if len(s) == 1 {
		fmt.Printf("%s\n", s[0])
		return
	}
	flags := getflags(s[1])

	stem := false
	for _,key := range flags {
		r, ok := rules[key]
		if ok && r.need {
			stem = true
			break
		}
	}
	if !stem {
		fmt.Printf("%s\n", s[0])
	}

	for _,key := range flags {
		//fmt.Fprintf(os.Stderr, "key: %v\n", key)
		r, ok := rules[key]
		if !ok {
			fmt.Fprintf(os.Stderr, "%s not found\n", key)
			continue
		}
		if r.need { // ignore this need flag
			continue
		}
		//fmt.Fprintf(os.Stderr, "%v\n", r)
		for _,a := range r.affix {
			x := s[0]
			if r.suffix {
				if a.cond == "" {
					if a.strip != "" {
						x = x[:len(x)-len(a.strip)]
					}
					expand(x + a.affix)
					break
				}

				if a.re.FindString(s[0]) == "" {
					continue
				}
				if a.strip != "" {
					x = x[:len(x)-len(a.strip)]
				}
				expand(fmt.Sprintf("%s\n", x+a.affix))
			} else {
				if a.cond == "" {
					if a.strip != "" {
						x = x[len(a.strip):]
					}
					expand(a.affix+x)
					break
				}
				if a.re.FindString(s[0]) == "" {
					continue
				}
				if a.strip != "" {
					x = x[len(a.strip):]
				}
				expand(a.affix+x)
			}
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
		fmt.Fprintf(os.Stderr, "%d words\n\n", count)
	}
	//words := make([]string, count)
	//affix := make([]string, count)
	for i := 0; ; i++ {
		l, _, err := d.ReadLine()
		if err  == io.EOF {
			break
		}
		line := string(l)
		if verbose {
			fmt.Fprintf(os.Stderr, "%d: %s\n",i, line)
		}
		expand(line)
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
