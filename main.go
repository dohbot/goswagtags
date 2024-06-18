package main

import (
	"bytes"
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
	"strings"
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

	for i := 0; i < flag.NArg(); i++ {
		path := flag.Arg(i)
		switch fi, err := os.Stat(path); {
		case err != nil:
			scanner.PrintError(os.Stderr, err)
		case fi.IsDir():
			if err := filepath.Walk(path, walkFunc); err != nil {
				scanner.PrintError(os.Stderr, err)
				os.Exit(1)
			}
		default:
			if err := process(path, *inPlace); err != nil {
				scanner.PrintError(os.Stderr, err)
				os.Exit(1)
			}
		}
	}
}

func walkFunc(path string, _ os.FileInfo, err error) error {
	if err == nil {
		err = process(path, *inPlace)
	}

	if err != nil {
		return err
	}

	return nil
}

func process(path string, inPlace bool) (err error) {
	if strings.HasSuffix(path, "_test.go") || strings.Contains(path, "/vendor/") {
		return
	}
	if !strings.HasSuffix(path, ".go") || strings.HasPrefix(path, ".") {
		return
	}

	var file *ast.File
	if file, err = parser.ParseFile(set, path, nil, parser.ParseComments); err != nil {
		return
	}

	if len(file.Comments) == 0 {
		file.Comments = []*ast.CommentGroup{{List: []*ast.Comment{{Slash: -1, Text: "//"}}}}
		defer func() { file.Comments = file.Comments[1:] }()
	}
	cmtMap := ast.NewCommentMap(set, file, file.Comments)
	skipped := make(map[ast.Node]bool)

	ast.Inspect(file, func(n ast.Node) bool {
		switch decl := n.(type) {
		case *ast.DeclStmt:
			skipped[decl.Decl] = true
		case *ast.GenDecl:
			applyStructNameTag(decl, skipped, cmtMap)
		default:
		}
		return true
	})

	file.Comments = cmtMap.Filter(file).Comments()

	var buf bytes.Buffer
	if err = format.Node(&buf, set, file); err != nil {
		return
	}

	out := buf.Bytes()
	out = tralingWsRegex.ReplaceAll(out, []byte(""))
	out = newlinesRegex.ReplaceAll(out, []byte("\n\n"))

	if inPlace {
		return os.WriteFile(path, out, defaultMode)
	}

	_, err = fmt.Fprintf(os.Stdout, "%s", out)
	return
}

func applyStructNameTag(decl *ast.GenDecl, skipped map[ast.Node]bool, cmtMap ast.CommentMap) {
	if decl.Tok != token.TYPE {
		return
	}

	spec := decl.Specs[0].(*ast.TypeSpec)
	if _, ok := spec.Type.(*ast.StructType); !ok {
		return
	}

	if skipped[decl] || !spec.Name.IsExported() {
		return
	}

	addNameTag(decl, spec)
	if cmtMap == nil {
		fmt.Println(cmtMap, decl.Doc.Text())
	}
	cmtMap[decl] = updateComment(cmtMap[decl], decl.Doc)
}

func addNameTag(decl *ast.GenDecl, spec *ast.TypeSpec) {
	name := spec.Name.Name
	text := fmt.Sprintf("@name %s", name)

	if spec.Comment == nil || !strings.HasPrefix(strings.TrimSpace(spec.Comment.Text()), text) {
		pos := spec.End()
		decl.Doc = &ast.CommentGroup{List: []*ast.Comment{{Slash: pos, Text: "//" + text}}}
		return
	}
}

func updateComment(groups []*ast.CommentGroup, doc *ast.CommentGroup) []*ast.CommentGroup {
	if doc == nil {
		return groups
	}

	var ret []*ast.CommentGroup
	hasInsert := false
	for _, group := range groups {
		if group.Pos() < doc.Pos() {
			ret = append(ret, group)
			continue
		}
		if group.Pos() == doc.Pos() {
			ret = append(ret, doc)
			hasInsert = true
			continue
		}
		if group.Pos() > doc.Pos() {
			if !hasInsert {
				ret = append(ret, doc)
				hasInsert = true
			}
			ret = append(ret, group)
			continue
		}
	}
	if !hasInsert {
		ret = append(ret, doc)
	}
	return ret
}
