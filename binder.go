package main

import (
	"code.google.com/p/snappy-go/snappy"
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func main() {
	var pkgName string
	flag.StringVar(&pkgName, "package", "main", "Name of the package")
	flag.Parse()

	fmt.Printf("package %s\n", pkgName)
	fmt.Println(`
import (
	"code.google.com/p/snappy-go/snappy"
	"encoding/base64"
	"log"
	"mime"
	"net/http"
	"path"
	"strings"
)`)
	fmt.Printf("\nconst (\n")

	re, _ := regexp.Compile("[^A-Za-z0-9]+")

	mp := make(map[string]string, 0)

	fkeys := make([]string, 0)
	for _, pattern := range flag.Args() {
		files, err := filepath.Glob(pattern)
		if err != nil {
			fmt.Errorf("Cannot glob files:\n%s\n", err)
		}
		sort.Strings(files)
		for _, f := range files {
			fi, err := os.Stat(f)
			if err == nil && !fi.IsDir() {

				fvar := "c" + strings.Replace(strings.Title(re.ReplaceAllString(filepath.Dir(f), " ")), " ", "", -1) + "_" + strings.Replace(strings.Title(re.ReplaceAllString(filepath.Base(f), " ")), " ", "", -1)
				fn := filepath.ToSlash(f)
				if _, ok := mp[fn]; !ok {
					mp[fn] = fvar
					fkeys = append(fkeys, fn)
					fmt.Printf("\t%s = \"", fvar)
					buf, err := ioutil.ReadFile(f)
					if err != nil {
						panic(err)
					}
					b, err := snappy.Encode(nil, buf)
					if err != nil {
						panic(err)
					}
					fmt.Print(base64.URLEncoding.EncodeToString(b))
					fmt.Printf("\"\n")
				}
			}
		}
	}

	fmt.Printf(")\n\n")

	fmt.Printf("var staticFiles = map[string]string{\n")
	for _, k := range fkeys {
		fmt.Printf("\t\"/%s\": %s,\n", k, mp[k])
	}
	fmt.Printf("}\n")

	fmt.Printf(`
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

	fmt.Println()

	fmt.Println(`
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
