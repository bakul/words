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

type affix interface {
	expand(string)
	String() string
}

type prefix struct {
	strip		string
	affix		string
	cond		string
	re		*regexp.Regexp
	morph		[]string
}

type suffix struct {
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
	affix		[]affix
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

func (a *prefix)String() string {
	s := fmt.Sprintf("  strip:%v affix:%v cond:%v", a.strip, a.affix, a.cond)
	if a.morph != nil {
		return s + " morph:" + fmt.Sprintf("%v", a.morph)
	}
	return s
}

func (a *suffix)String() string {
	s := fmt.Sprintf("  strip:%v affix:%v cond:%v", a.strip, a.affix, a.cond)
	if a.morph != nil {
		return s + " morph:" + fmt.Sprintf("%v", a.morph)
	}
	return s
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
		line, err := a.r.ReadString('\n')
		if err == io.EOF {
			return nil
		}
		a.ln++
		arg := spaceRE.FindAllString(strings.TrimSpace(line), -1)
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

func newPrefix(strip string, aff string, cond string) (*prefix, error) {
	var err error
	var re *regexp.Regexp
	if cond != "." {
		re, err = regexp.Compile("^"+cond+".*")
		if err != nil {
			return nil, err
		}
	} else {
		cond = ""
	}
	return &prefix{strip:strip, affix: aff, cond: cond, re: re}, nil
}


func newSuffix(strip string, aff string, cond string) (*suffix, error) {
	var err error
	var re *regexp.Regexp
	if cond != "." {
		re, err = regexp.Compile(".*"+cond+"$")
		if err != nil {
			return nil, err
		}
	} else {
		cond = ""
	}
	return &suffix{strip:strip, affix: aff, cond: cond, re: re}, nil
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
		r = &rule{name:flag,ln:a.ln, affix:make([]affix, count)}
		if arg[2] == "Y" {
			r.cross = true
		}
		var suffix bool
		if arg[0] == "SFX" {
			suffix = true
		}
		for i := range r.affix {
			arg := a.getRule()
			if arg == nil {
				fmt.Fprintf(os.Stderr, "%s: file too short\n", flag)
				panic("bad affix file")
			}
			if arg[2] == "0" {
				arg[2] = ""
			}
			if len(arg) < 5 {
				//fmt.Fprintf(os.Stderr, "%d: not enough fields\n", a.ln)

				arg = append(arg, ".")
			}
			var affix affix
			if suffix {
				affix, err = newSuffix(arg[2], arg[3], arg[4])
			} else {
				affix, err = newPrefix(arg[2], arg[3], arg[4])
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "%d: %v\n", a.ln, err)
			}
			r.affix[i] = affix
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

func (a *suffix) expand(word string) {
	if a.cond == "" {
		if a.strip != "" {
			word = word[:len(word)-len(a.strip)]
		}
		expandAll(word + a.affix)
		return
	}

	if a.re.FindString(word) == "" {
		return
	}
	if a.strip != "" {
		word = word[:len(word)-len(a.strip)]
	}
	expandAll(fmt.Sprintf("%s\n", word+a.affix))
}

func (a *prefix) expand(word string) {
	if a.cond == "" {
		if a.strip != "" {
			word = word[len(a.strip):]
		}
		expandAll(a.affix+word)
		return
	}
	if a.re.FindString(word) == "" {
		return
	}
	if a.strip != "" {
		word = word[len(a.strip):]
	}
	expandAll(a.affix+word)
}

var ln = 2

func expandAll(line string) {
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
			fmt.Fprintf(os.Stderr, "%d: %s not found\n", ln, key)
			continue
		}
		if r.need { // ignore this need flag
			continue
		}
		//fmt.Fprintf(os.Stderr, "%v\n", r)
		for _,a := range r.affix {
			a.expand(s[0])
		}
	}
}

func processDict(f *os.File) error {
	d := bufio.NewReader(f)
	line, err := d.ReadString('\n')
	if err != nil {
		return err
	}
	count, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil {
		panic(fmt.Sprintf("error: %v", err.Error()))
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "%d words\n\n", count)
	}
	//words := make([]string, count)
	//affix := make([]string, count)
	for i := 0; ; i++ {
		line, err := d.ReadString('\n')
		if err  == io.EOF {
			break
		}
		line = strings.TrimSpace(line)
		if verbose {
			fmt.Fprintf(os.Stderr, "%d: |%s|\n",i, line)
		}
		expandAll(line)
		ln++
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
