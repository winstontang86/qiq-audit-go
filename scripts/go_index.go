package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type functionInfo struct {
	Name        string `json:"name"`
	Receiver    string `json:"receiver,omitempty"`
	LineStart   int    `json:"line_start"`
	LineEnd     int    `json:"line_end"`
	ColumnStart int    `json:"column_start"`
	ColumnEnd   int    `json:"column_end"`
	BodyLines   int    `json:"body_lines"`
	InputCount  int    `json:"input_count"`
	ResultCount int    `json:"result_count"`
	start       token.Pos
	end         token.Pos
}

type symbolInfo struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Function  string `json:"function,omitempty"`
	LineStart int    `json:"line_start"`
	LineEnd   int    `json:"line_end"`
	start     token.Pos
}

type locationInfo struct {
	Line       int      `json:"line"`
	Symbol     string   `json:"symbol,omitempty"`
	Ambiguous  bool     `json:"ambiguous,omitempty"`
	Candidates []string `json:"candidates,omitempty"`
}

type fileInfo struct {
	File        string         `json:"file"`
	Package     string         `json:"package"`
	PackageLine int            `json:"package_line"`
	LineCount   int            `json:"line_count"`
	Generated   bool           `json:"generated"`
	Functions   []functionInfo `json:"functions"`
	Symbols     []symbolInfo   `json:"symbols"`
	Location    *locationInfo  `json:"location,omitempty"`
	Error       string         `json:"error,omitempty"`
}

func physicalLines(src []byte) int {
	if len(src) == 0 {
		return 0
	}
	lines := bytes.Count(src, []byte{'\n'})
	if src[len(src)-1] != '\n' {
		lines++
	}
	return lines
}

func fieldCount(fields *ast.FieldList) int {
	if fields == nil {
		return 0
	}
	count := 0
	for _, field := range fields.List {
		if len(field.Names) == 0 {
			count++
		} else {
			count += len(field.Names)
		}
	}
	return count
}

func enclosing(functions []functionInfo, pos token.Pos) *functionInfo {
	var closest *functionInfo
	for i := range functions {
		fn := &functions[i]
		if fn.start < pos && pos < fn.end && (closest == nil || fn.start > closest.start) {
			closest = fn
		}
	}
	return closest
}

func expressionText(fset *token.FileSet, expr ast.Expr) string {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, expr); err != nil {
		return "<unknown>"
	}
	return buf.String()
}

func indexSource(path string, src []byte, queryLine int) fileInfo {
	result := fileInfo{
		File: path, LineCount: physicalLines(src),
		Functions: []functionInfo{}, Symbols: []symbolInfo{},
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments|parser.AllErrors)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	line := func(pos token.Pos) int { return fset.PositionFor(pos, false).Line }
	result.Package = file.Name.Name
	result.PackageLine = line(file.Package)
	for _, group := range file.Comments {
		if group.Pos() > file.Package {
			break
		}
		for _, comment := range group.List {
			text := comment.Text
			if strings.HasPrefix(text, "// Code generated ") && strings.HasSuffix(text, " DO NOT EDIT.") {
				result.Generated = true
			}
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		var name, receiver string
		var typ *ast.FuncType
		var body *ast.BlockStmt
		switch n := node.(type) {
		case *ast.FuncDecl:
			name, typ, body = n.Name.Name, n.Type, n.Body
			if n.Recv != nil && len(n.Recv.List) > 0 {
				receiver = expressionText(fset, n.Recv.List[0].Type)
				if strings.HasPrefix(receiver, "*") {
					name = "(" + receiver + ")." + name
				} else {
					name = receiver + "." + name
				}
			}
		case *ast.FuncLit:
			typ, body = n.Type, n.Body
			name = fmt.Sprintf("func@L%dC%d", line(n.Pos()), fset.PositionFor(n.Pos(), false).Column)
			if parent := enclosing(result.Functions, n.Pos()); parent != nil {
				name = parent.Name + "." + name
			} else {
				name = "<package " + result.Package + ">." + name
			}
		default:
			return true
		}
		bodyLines := 0
		if body != nil {
			bodyLines = line(body.Rbrace) - line(body.Lbrace) - 1
			if bodyLines < 0 {
				bodyLines = 0
			}
		}
		result.Functions = append(result.Functions, functionInfo{
			Name: name, Receiver: receiver, LineStart: line(node.Pos()), LineEnd: line(node.End()),
			BodyLines: bodyLines, InputCount: fieldCount(typ.Params), ResultCount: fieldCount(typ.Results),
			ColumnStart: fset.PositionFor(node.Pos(), false).Column,
			ColumnEnd:   fset.PositionFor(node.End(), false).Column,
			start:       node.Pos(), end: node.End(),
		})
		return true
	})
	addSymbol := func(name, kind string, node ast.Node) {
		result.Symbols = append(result.Symbols, symbolInfo{
			Name: name, Kind: kind, LineStart: line(node.Pos()), LineEnd: line(node.End()), start: node.Pos(),
		})
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.GenDecl:
			for _, spec := range n.Specs {
				if value, ok := spec.(*ast.ValueSpec); ok {
					for _, ident := range value.Names {
						addSymbol(ident.Name, n.Tok.String(), value)
					}
				}
			}
		case *ast.TypeSpec:
			addSymbol(n.Name.Name, "type", n)
			var fields *ast.FieldList
			kind := "field"
			switch typ := n.Type.(type) {
			case *ast.StructType:
				fields = typ.Fields
			case *ast.InterfaceType:
				fields, kind = typ.Methods, "interface_method"
			default:
				fields = nil
			}
			if fields != nil {
				for _, field := range fields.List {
					for _, ident := range field.Names {
						addSymbol(n.Name.Name+"."+ident.Name, kind, field)
					}
				}
			}
		default:
			return true
		}
		return true
	})
	for i := range result.Symbols {
		if parent := enclosing(result.Functions, result.Symbols[i].start); parent != nil {
			result.Symbols[i].Function = parent.Name
		}
	}
	if queryLine > 0 {
		if queryLine > result.LineCount {
			result.Error = fmt.Sprintf("line %d exceeds file length %d", queryLine, result.LineCount)
			return result
		}
		symbol := "<file>"
		if queryLine == result.PackageLine {
			symbol = "<package " + result.Package + ">"
		}
		candidates := []string{}
		for _, fn := range result.Functions {
			if fn.LineStart <= queryLine && queryLine <= fn.LineEnd {
				candidates = append(candidates, fn.Name)
			}
		}
		if len(candidates) == 1 {
			symbol = candidates[0]
		} else if len(candidates) > 1 {
			result.Location = &locationInfo{
				Line: queryLine, Ambiguous: true, Candidates: candidates,
			}
			return result
		} else {
			width := result.LineCount + 1
			for _, decl := range result.Symbols {
				if decl.LineStart <= queryLine && queryLine <= decl.LineEnd && decl.LineEnd-decl.LineStart < width {
					symbol, width = decl.Name, decl.LineEnd-decl.LineStart
				}
			}
		}
		result.Location = &locationInfo{
			Line: queryLine, Symbol: symbol,
		}
	}
	return result
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	flags := flag.NewFlagSet("go_index", flag.ContinueOnError)
	flags.SetOutput(stderr)
	queryLine := flags.Int("line", 0, "locate a physical source line in exactly one file")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	paths := flags.Args()
	if len(paths) == 0 || *queryLine < 0 || (*queryLine > 0 && len(paths) != 1) {
		_, _ = fmt.Fprintln(stderr, "usage: go run go_index.go -- [-line N] /absolute/file.go ...")
		return 2
	}
	results := make([]fileInfo, 0, len(paths))
	code := 0
	for _, path := range paths {
		absolute, err := filepath.Abs(path)
		if err == nil {
			path = absolute
		}
		result := fileInfo{
			File: path, Functions: []functionInfo{}, Symbols: []symbolInfo{},
		}
		if err == nil {
			var info os.FileInfo
			info, err = os.Lstat(path)
			if err == nil && (!info.Mode().IsRegular() || filepath.Ext(path) != ".go") {
				err = fmt.Errorf("expected a regular .go file, not a directory or symlink")
			}
		}
		if err == nil {
			var src []byte
			src, err = os.ReadFile(path)
			if err == nil {
				result = indexSource(path, src, *queryLine)
			}
		}
		if err != nil {
			result.Error = err.Error()
		}
		if result.Error != "" {
			code = 1
		}
		results = append(results, result)
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(results); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	return code
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
