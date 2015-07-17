package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/golang/snappy"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func main() {
	var (
		pkgName string
		recurse bool
		outFile string
	)
	flag.StringVar(&pkgName, "package", "main", "Name of the package")
	flag.StringVar(&outFile, "o", "", "Output file (otherwise writes to stdout)")
	flag.BoolVar(&recurse, "r", false, "Recursively add all files in subfolders")
	flag.Parse()

	var out *os.File
	var err error
	if outFile != "" {
		out, err = os.Create(outFile)
		if err != nil {
			panic(err)
		}
		defer out.Close()
	} else {
		out = os.Stdout
	}

	fmt.Fprintf(out, "package %s\n", pkgName)
	fmt.Fprintln(out, `
import (
	"github.com/golang/snappy/snappy"
	"encoding/base64"
	"log"
	"mime"
	"net/http"
	"path"
	"strings"
)`)
	fmt.Fprintf(out, "\nconst (\n")

	re, _ := regexp.Compile("[^A-Za-z0-9]+") // regular expression to replace unwanted file characters
	mp := make(map[string]string, 0)         // map to track filenames to the constant defining the data
	fv := make(map[string]bool, 0)           // map to track if the name of the constant is already used and needs to be incremented
	fkeys := make([]string, 0)               // list of filename keys (for sorting)

	// define the "addfile" function
	addfile := func(f string) error {
		fvar := "c" + strings.Replace(strings.Title(re.ReplaceAllString(filepath.Dir(f), " ")), " ", "", -1) + "_" + strings.Replace(strings.Title(re.ReplaceAllString(filepath.Base(f), " ")), " ", "", -1)
		fn := filepath.ToSlash(f)
		if _, ok := mp[fn]; !ok {
			gvar := fvar
			i := 1
			for {
				if _, ok := fv[gvar]; ok {
					gvar = fmt.Sprintf("%s%d", fvar, i)
					i++
				} else {
					break
				}
			}
			fvar = gvar
			fv[fvar] = true
			mp[fn] = fvar
			fkeys = append(fkeys, fn)
			fmt.Fprintf(out, "\t%s = \"", fvar)
			buf, err := ioutil.ReadFile(f)
			if err != nil {
				return err
			}
			b := snappy.Encode(nil, buf)
			fmt.Fprint(out, base64.URLEncoding.EncodeToString(b))
			fmt.Fprintf(out, "\"\n")
		}
		return nil
	}

	for _, pattern := range flag.Args() {
		files, err := filepath.Glob(pattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cannot glob files:\n%s\n", err)
		}
		sort.Strings(files)
		for _, f := range files {
			fi, err := os.Stat(f)
			if err == nil {
				if !strings.HasPrefix(fi.Name(), ".") {
					if !fi.IsDir() {
						err = addfile(f)
						if err != nil {
							panic(err)
						}
					} else if recurse {
						err = filepath.Walk(f, func(path string, info os.FileInfo, err error) error {
							if err != nil {
								return err
							}
							if info.IsDir() {
								return nil
							}
							return addfile(path)
						})
					}
				}
			}
		}
	}

	fmt.Fprintf(out, ")\n\n")

	fmt.Fprintf(out, "var staticFiles = map[string]string{\n")
	for _, k := range fkeys {
		fmt.Fprintf(out, "\t\"/%s\": %s,\n", k, mp[k])
	}
	fmt.Fprintf(out, "}\n")

	fmt.Fprintf(out, `
func Lookup(path string) []byte {
	s, ok := staticFiles[path]
	if !ok {
		return nil
	} else {
		d, err := base64.URLEncoding.DecodeString(s)
		if err != nil {
			log.Print("%s.Lookup: ", err)
			return nil
		}
		r, err := snappy.Decode(nil, d)
		if err != nil {
			log.Print("%s.Lookup: ", err)
			return nil
		}
		return r
	}
}`, pkgName, pkgName)

	fmt.Fprintln(out)

	fmt.Fprintln(out, `
func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	if strings.HasSuffix(p, "/") {
		p += "index.html"
	}
	b := Lookup(p)
	if b != nil {
		mt := mime.TypeByExtension(path.Ext(p))
		if mt != "" {
			w.Header().Set("Content-Type", mt)
		}
		w.Write(b)
	} else {
		http.NotFound(w, r)
	}
}`)
}
