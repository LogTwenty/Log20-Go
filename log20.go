package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var count = 1

func main() {
	log.SetFlags(0)
	log.SetPrefix("gocyclo: ")
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		os.Exit(2)
	}

	var counterMutex = &sync.Mutex{}
	BasicBlocks := analyze(args, counterMutex)
	writeBasicBlocks(os.Stdout, BasicBlocks)
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
	for _, BasicBlock := range sortedBasicBlocks {
		fmt.Fprintln(w, BasicBlock)
	}
	return len(sortedBasicBlocks)
}

// BasicBlock should have a descriptive comment
type BasicBlock struct {
	PkgName    string
	FuncName   string
	Complexity int
	Pos        token.Position
	EndPos     token.Position
	ID         int
	// MethodSignature string
	// BasicBlockID    int32
	NumTrace        int32
	NumDebug        int32
	NumInfo         int32
	NumWarn         int32
	NumError        int32
	NumFatal        int32
	beginLineNo     int32
	endLineNo       int32
	predIds         []int32
	succIds         []int32
}


func (s BasicBlock) String() string {
	return fmt.Sprintf("%d %s %s %s %s %d", s.Complexity, s.PkgName, s.FuncName, s.Pos, s.EndPos, s.ID)
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

func calculateShannonsEntropy(probabilities []float64) float64 {
	var sum float64
	for _, probability := range probabilities {
		sum += (probability * (math.Log2(float64(probability))))
	}
	return -sum
}

func calculateProbablityOfSpecificLog(probabilities []float64) float64 {
	var sum float64
	for _, probability := range probabilities {
		sum += (probability * (math.Log2(float64(probability))))
	}
	return -sum
}