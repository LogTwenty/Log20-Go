// Copyright 2013 Frederik Zipp. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Gocyclo calculates the cyclomatic complexities of functions and
// methods in Go source code.
//
// Usage:
//      gocyclo [<flag> ...] <Go file or directory> ...
//
// Flags:
//      -over N   show functions with complexity > N only and
//                return exit code 1 if the output is non-empty
//      -top N    show the top N most complex functions only
//      -avg      show the average complexity
//
// The output fields for each line are:
// <complexity> <package> <function> <file:row:column>
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
  "sync"
)

const usageDoc = `Calculate cyclomatic complexities of Go functions.
Usage:
        gocyclo [flags] <Go file or directory> ...

Flags:
        -over N   show functions with complexity > N only and
                  return exit code 1 if the set is non-empty
        -top N    show the top N most complex functions only
        -avg      show the average complexity over all functions,
                  not depending on whether -over or -top are set

The output fields for each line are:
<complexity> <package> <function> <file:row:column>
`

func usage() {
	fmt.Fprintf(os.Stderr, usageDoc)
	os.Exit(2)
}

var (
	over = flag.Int("over", 0, "show functions with complexity > N only")
	top  = flag.Int("top", -1, "show the top N most complex functions only")
	avg  = flag.Bool("avg", false, "show the average complexity")
  count = 1
)
func main() {
	log.SetFlags(0)
	log.SetPrefix("gocyclo: ")
	flag.Usage = usage
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		usage()
	}

  var counterMutex = &sync.Mutex{}
	BasicBlocks := analyze(args, counterMutex)
	sort.Sort(byComplexity(BasicBlocks))
	written := writeBasicBlocks(os.Stdout, BasicBlocks)

	if *avg {
		showAverage(BasicBlocks)
	}

	if *over > 0 && written > 0 {
		os.Exit(1)
	}
}

func analyze(paths []string, counterMutex *sync.Mutex) []BasicBlock {
	var BasicBlocks []BasicBlock
	for _, path := range paths {
		if isDir(path) {
			BasicBlocks = analyzeDir(path, BasicBlocks, counterMutex)
		} else {
			BasicBlocks = analyzeFile(path, BasicBlocks, counterMutex)
		}
	}
	return BasicBlocks
}

func isDir(filename string) bool {
	fi, err := os.Stat(filename)
	return err == nil && fi.IsDir()
}

func analyzeFile(fname string, BasicBlocks []BasicBlock, counterMutex *sync.Mutex) []BasicBlock {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fname, nil, 0)
	if err != nil {
		log.Fatal(err)
	}
	return buildBasicBlocks(f, fset, BasicBlocks, counterMutex)
}

func analyzeDir(dirname string, BasicBlocks []BasicBlock, counterMutex *sync.Mutex) []BasicBlock {
	filepath.Walk(dirname, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(path, ".go") {
			BasicBlocks = analyzeFile(path, BasicBlocks, counterMutex)
		}
		return err
	})
	return BasicBlocks
}

func writeBasicBlocks(w io.Writer, sortedBasicBlocks []BasicBlock) int {
	for i, BasicBlock := range sortedBasicBlocks {
		if i == *top {
			return i
		}
		if BasicBlock.Complexity <= *over {
			return i
		}
		fmt.Fprintln(w, BasicBlock)
	}
	return len(sortedBasicBlocks)
}

func showAverage(BasicBlocks []BasicBlock) {
	fmt.Printf("Average: %.3g\n", average(BasicBlocks))
}

func average(BasicBlocks []BasicBlock) float64 {
	total := 0
	for _, s := range BasicBlocks {
		total += s.Complexity
	}
	return float64(total) / float64(len(BasicBlocks))
}

type BasicBlock struct {
	PkgName    string
	FuncName   string
	Complexity int
	Pos        token.Position
	EndPos     token.Position
  ID int
}

func (s BasicBlock) String() string {
	return fmt.Sprintf("%d %s %s %s %s %d", s.Complexity, s.PkgName, s.FuncName, s.Pos, s.EndPos, s.ID)
}

type byComplexity []BasicBlock

func (s byComplexity) Len() int      { return len(s) }
func (s byComplexity) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s byComplexity) Less(i, j int) bool {
	return s[i].Complexity >= s[j].Complexity
}

func buildBasicBlocks(f *ast.File, fset *token.FileSet, BasicBlocks []BasicBlock, counterMutex *sync.Mutex) []BasicBlock {
	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
      counterMutex.Lock()
			BasicBlocks = append(BasicBlocks, BasicBlock{
				PkgName:    f.Name.Name,
				FuncName:   funcName(fn),
				Complexity: complexity(fn),
				Pos:        fset.Position(fn.Pos()),
				EndPos:     fset.Position(fn.End()),
        ID:         count,
			})
      count++
      counterMutex.Unlock()
		}
	}
	return BasicBlocks
}

// funcName returns the name representation of a function or method:
// "(Type).Name" for methods or simply "Name" for functions.
func funcName(fn *ast.FuncDecl) string {
	if fn.Recv != nil {
		if fn.Recv.NumFields() > 0 {
			typ := fn.Recv.List[0].Type
			return fmt.Sprintf("(%s).%s", recvString(typ), fn.Name)
		}
	}
	return fn.Name.Name
}

// recvString returns a string representation of recv of the
// form "T", "*T", or "BADRECV" (if not a proper receiver type).
func recvString(recv ast.Expr) string {
	switch t := recv.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + recvString(t.X)
	}
	return "BADRECV"
}

// complexity calculates the cyclomatic complexity of a function.
func complexity(fn *ast.FuncDecl) int {
	v := complexityVisitor{}
	ast.Walk(&v, fn)
	return v.Complexity
}

type complexityVisitor struct {
	// Complexity is the cyclomatic complexity
	Complexity int
}

// Visit implements the ast.Visitor interface.
func (v *complexityVisitor) Visit(n ast.Node) ast.Visitor {
	switch n := n.(type) {
	case *ast.FuncDecl, *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.CaseClause, *ast.CommClause:
		v.Complexity++
	case *ast.BinaryExpr:
		if n.Op == token.LAND || n.Op == token.LOR {
			v.Complexity++
		}
  // We might also need to check Go/Defer statements
	case *ast.CallExpr:
    fmt.Printf("%+v\n", n.Fun)
	}
	return v
}
