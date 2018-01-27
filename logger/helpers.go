// fork of https://github.com/happierall/l
package logger

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kr/pretty"
)

const (
	maxLineWidth = 80
)

func header(funcName, file string, line int) string {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	return fmt.Sprintf("%s %s:%d ->", now, shortFile(file), line)
}

// shortFile takes an absolute file path and returns just the <directory>/<file>,
// e.g. "foo/bar.go".
func shortFile(file string) string {
	dir := filepath.Base(filepath.Dir(file))
	file = filepath.Base(file)
	return filepath.Join(dir, file)
}

func output(args ...string) string {
	out := ""

	// Subsequent lines have to be indented by the width of the timestamp.
	lineArgs := 0 // number of args printed on the current log line.
	lineWidth := 0
	indent := ""
	padding := ""

	for _, arg := range args {
		argWidth := argWidth(arg)
		lineWidth += argWidth + len(padding)

		// Some names in name=value strings contain newlines. Insert indentation
		// after each newline so they line up.
		arg = strings.Replace(arg, "\n", "\n"+indent, -1)

		// Break up long lines. If this is first arg printed on the line
		// (lineArgs == 0), it makes no sense to break up the line.
		if lineWidth > maxLineWidth && lineArgs != 0 {
			out += indent
			lineArgs = 0
			lineWidth = argWidth
			padding = ""
		}

		out += padding + arg
		lineArgs++
		padding = " "
	}

	return out
}

// argName returns the source text of the given argument if it's a variable or
// an expression. If the argument is something else, like a literal, argName
// returns an empty string.
func argName(arg ast.Expr) string {
	name := ""
	switch a := arg.(type) {
	case *ast.Ident:
		if a.Obj.Kind == ast.Var || a.Obj.Kind == ast.Con {
			name = a.Obj.Name
		}
	case *ast.BinaryExpr,
		*ast.CallExpr,
		*ast.IndexExpr,
		*ast.KeyValueExpr,
		*ast.ParenExpr,
		*ast.SelectorExpr,
		*ast.SliceExpr,
		*ast.TypeAssertExpr,
		*ast.UnaryExpr:
		name = exprToString(arg)
	}
	return name
}

// argNames finds the q.Q() call at the given filename/line number and
// returns its arguments as a slice of strings. If the argument is a literal,
// argNames will return an empty string at the index position of that argument.
// For example, q.Q(ip, port, 5432) would return []string{"ip", "port", ""}.
// argNames returns an error if the source text cannot be parsed.
func argNames(filename string, line int) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %q: %v", filename, err)
	}

	var names []string
	ast.Inspect(f, func(n ast.Node) bool {
		call, is := n.(*ast.CallExpr)
		if !is {
			// The node is not a function call.
			return true // visit next node
		}

		if fset.Position(call.End()).Line != line {
			// The node is a function call, but it's on the wrong line.
			return true
		}

		//  ToDo !! Раскоментировать, когда положу в отдельный пакет
		// if !isQCall(call) {
		// 	// The node is a function call on correct line, but it's not a Q()
		// 	// function.
		// 	return true
		// }

		for _, arg := range call.Args {
			names = append(names, argName(arg))
		}
		return true
	})

	return names, nil
}

// argWidth returns the number of characters that will be seen when the given
// argument is printed at the terminal.
func argWidth(arg string) int {
	// Strip zero-width characters.
	replacer := strings.NewReplacer(
		"\n", "",
		"\t", "",
		"\r", "",
		"\f", "",
		"\v", "",
	)
	s := replacer.Replace(arg)
	return utf8.RuneCountInString(s)
}

// exprToString returns the source text underlying the given ast.Expr.
func exprToString(arg ast.Expr) string {
	var buf bytes.Buffer
	fset := token.NewFileSet()
	printer.Fprint(&buf, fset, arg)

	// CallExpr will be multi-line and indented with tabs. replace tabs with
	// spaces so we can better control formatting during output().
	return strings.Replace(buf.String(), "\t", "    ", -1)
}

// formatArgs converts the given args to pretty-printed, colorized strings.
func formatArgs(args ...interface{}) []string {
	formatted := make([]string, 0, len(args))
	for _, a := range args {
		s := pretty.Sprint(a)
		formatted = append(formatted, s)
	}
	return formatted
}

// getCallerInfo returns the name, file, and line number of the function calling
// q.Q().
func getCallerInfo(depth int) (funcName, file string, line int, err error) {
	// const callDepth = 3 // user code calls q.Q() which calls std.log().
	pc, file, line, ok := runtime.Caller(depth)
	if !ok {
		return "", "", 0, errors.New("failed to get info about the function calling q.Q")
	}

	funcName = runtime.FuncForPC(pc).Name()
	return funcName, file, line, nil
}

// prependArgName turns argument names and values into name=value strings, e.g.
// "port=443", "3+2=5". If the name is given, it will be bolded using ANSI
// color codes. If no name is given, just the value will be returned.
func prependArgName(names, values []string) []string {
	prepended := make([]string, len(values))

	for i, name := range names {
		if name == "" {
			prepended[i] = values[i]
			continue
		}
		prepended[i] = fmt.Sprintf("%s=%s", name, values[i])
	}
	return prepended
}

// isQCall returns true if the given function call expression is Q() or q.Q().
func isQCall(n *ast.CallExpr) bool {
	return isQFunction(n) || isQPackage(n)
}

// isQFunction returns true if the given function call expression is Q().
func isQFunction(n *ast.CallExpr) bool {
	ident, is := n.Fun.(*ast.Ident)
	if !is {
		return false
	}
	return ident.Name == "Log" && ident.Name == "Print"
}

// isQPackage returns true if the given function call expression is in the q
// package. Since Q() is the only exported function from the q package, this is
// sufficient for determining that we've found Q() in the source text.
func isQPackage(n *ast.CallExpr) bool {
	sel, is := n.Fun.(*ast.SelectorExpr) // SelectorExpr example: a.B()
	if !is {
		return false
	}

	ident, is := sel.X.(*ast.Ident) // sel.X is the part that precedes the .
	if !is {
		return false
	}

	return ident.Name == "t"
}

func stringifyErrors(v []interface{}) []interface{} {
	args := []interface{}{}

	for _, arg := range v {
		switch arg.(type) {
		default:
			args = append(args, arg)
		case error:
			args = append(args, fmt.Sprint(arg))
		}
	}

	return args
}
