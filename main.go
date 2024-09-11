package main

import (
	"bytes"
	"cmp"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/scanner"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/iancoleman/strcase"
)

var (
	set            = token.NewFileSet()
	tralingWsRegex = regexp.MustCompile(`(?m)[\t ]+$`)
	newlinesRegex  = regexp.MustCompile(`(?m)\n{3,}`)
	defaultMode    = os.FileMode(0644)
)

var (
	inPlace = flag.Bool("i", false, "Make in-place editing")
)

func main() {
	flag.Parse()

	if flag.NArg() == 0 {
		_, _ = fmt.Fprintf(os.Stderr, "usage: goswagtags [flags] [path ...]\n")
		flag.PrintDefaults()
		return
	}

	skipExp := regexp.MustCompile(`(^\.+)[^/]*|(_test\.go$)|(^.*/vender/.*$)`)
	for i := 0; i < flag.NArg(); i++ {
		path := flag.Arg(i)
		if !strings.HasSuffix(path, ".go") || skipExp.MatchString(path) {
			continue
		}

		var err error
		switch stat, e := os.Stat(path); {
		case e != nil:
			scanner.PrintError(os.Stderr, e)
		case stat.IsDir():
			err = filepath.Walk(path, walk)
		default:
			err = walk(path, stat, e)
		}

		if err != nil {
			scanner.PrintError(os.Stderr, err)
			os.Exit(1)
		}
	}

}
func walk(path string, _ os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	return process(path)
}

func process(path string) (err error) {

	var file *ast.File
	if file, err = parser.ParseFile(set, path, nil, parser.ParseComments); err != nil {
		return
	}

	comments := []*ast.CommentGroup{{List: []*ast.Comment{{Slash: -1, Text: "//"}}}}
	for _, groups := range file.Comments {
		if !strings.HasPrefix(groups.Text(), "@name ") {
			comments = append(comments, groups)
		}
	}

	funcMap := make(map[token.Pos]*ast.FuncDecl)

	ast.Inspect(file, func(n ast.Node) bool {
		switch decl := n.(type) {
		case *ast.FuncDecl:
			funcMap[decl.End()] = decl
		case *ast.GenDecl:
			applyStructNameTag(decl, funcMap, &comments)
		default:
		}
		return true
	})

	slices.SortFunc(comments, func(a, b *ast.CommentGroup) int {
		if r := cmp.Compare(a.Pos(), b.Pos()); r != 0 {
			return r
		}
		return cmp.Compare(a.End(), b.End())
	})

	if comments[0].Pos() == -1 {
		comments = comments[1:]
	}
	file.Comments = comments

	var buf bytes.Buffer
	if err = format.Node(&buf, set, file); err != nil {
		return
	}

	out := buf.Bytes()
	out = tralingWsRegex.ReplaceAll(out, []byte(""))
	out = newlinesRegex.ReplaceAll(out, []byte("\n\n"))

	if *inPlace {
		return os.WriteFile(path, out, defaultMode)
	}

	_, err = fmt.Fprintf(os.Stdout, "%s", out)
	return
}

func applyStructNameTag(decl *ast.GenDecl, funcMap map[token.Pos]*ast.FuncDecl, comments *[]*ast.CommentGroup) {
	if decl == nil || decl.Tok != token.TYPE || len(decl.Specs) < 1 {
		return
	}

	spec := decl.Specs[0].(*ast.TypeSpec)
	if _, ok := spec.Type.(*ast.StructType); !ok {
		return
	}

	name := spec.Name.String()

	for _, f := range funcMap {
		if f.Pos() < decl.Pos() && decl.Pos() < f.End() {
			name = strcase.ToCamel(f.Name.String() + "_" + name)
			break
		}

	}

	text := fmt.Sprintf("@name %s", name)
	decl.Doc = &ast.CommentGroup{List: []*ast.Comment{{Slash: spec.End(), Text: "//" + text}}}
	*comments = append(*comments, decl.Doc)
}
