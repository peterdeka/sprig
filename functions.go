/*
Sprig: Template functions for Go.

This package contains a number of utility functions for working with data
inside of Go `html/template` and `text/template` files.

To add these functions, use the `template.Funcs()` method:

	t := templates.New("foo").Funcs(sprig.FuncMap())

Note that you should add the function map before you parse any template files.

	In several cases, Sprig reverses the order of arguments from the way they
	appear in the standard library. This is to make it easier to pipe
	arguments into functions.

Date Functions

	- date: Format a date, where a date is an integer type or a time.Time type, and
	  format is a time.Format formatting string.
	- date_modify: Given a date, modify it with a duration: `date_modify "-1.5h" now`. If the duration doesn't
	parse, it returns the time unaltered. See `time.ParseDuration` for info on duration strings.
	- now: Current time.Time, for feeding into date-related functions.

String Functions

	- trim: strings.TrimSpace
	- trimall: strings.Trim, but with the argument order reversed `trimall "$" "$5.00"` or `"$5.00 | trimall "$"`
	- upper: strings.ToUpper
	- lower: strings.ToLower
	- title: strings.Title
	- repeat: strings.Repeat, but with the arguments switched: `repeat count str`. (This simplifies common pipelines)
	- substr: Given string, start, and length, return a substr.

String Slice Functions:

	- join: strings.Join, but as `join SEP SLICE`
	- split: strings.Split, but as `split SEP STRING`. The results are returned
	  as a map with the indexes set to _N, where N is an integer starting from 0.
	  Use it like this: `{{$v := "foo/bar/baz" | split "/"}}{{$v._0}}` (Prints `foo`)

Conversions:

	- atoi: Convert a string to an integer. 0 if the integer could not be parsed.

Defaults:

	- default: Give a default value. Used like this: trim "   "| default "empty".
	  Since trim produces an empty string, the default value is returned. For
	  things with a length (strings, slices, maps), len(0) will trigger the default.
	  For numbers, the value 0 will trigger the default. For booleans, false will
	  trigger the default. For structs, the default is never returned (there is
	  no clear empty condition). For everything else, nil value triggers a default.

Reflection:

	- typeOf: Takes an interface and returns a string representation of the type.
	  For pointers, this will return a type prefixed with an asterisk(`*`). So
	  a pointer to type `Foo` will be `*Foo`.
	- typeIs: Compares an interface with a string name, and returns true if they match.
	  Note that a pointer will not match a reference. For example `*Foo` will not
	  match `Foo`.
	- typeIsLike: Compares an interface with a string name and returns true if
	  the interface is that `name` or that `*name`. In other words, if the given
	  value matches the given type or is a pointer to the given type, this returns
	  true.
	- kindOf: Takes an interface and returns a string representation of its kind.
	- kindIs: Returns true if the given string matches the kind of the given interface.

	Note: None of these can test whether or not something implements a given
	interface, since doing so would require compiling the interface in ahead of
	time.


Math Functions:

	- add1: Increment an integer by 1
	- add: Sum two integers
	- sub: Subtract the second integer from the first
	- div: Divide the first integer by the second
	- mod: Module of first integer divided by second
	- mul: Multiply two integers
	- biggest: Return the biggest of two integers

REMOVED (implemented in Go 1.2)

	- gt: Greater than (integer)
	- lt: Less than (integer)
	- gte: Greater than or equal to (integer)
	- lte: Less than or equal to (integer)

*/
package sprig

import (
	"fmt"
	"html/template"
	"reflect"
	"strconv"
	"strings"
	ttemplate "text/template"
	"time"
)

// Produce the function map.
//
// Use this to pass the functions into the template engine:
//
// 	tpl := template.New("foo").Funcs(sprig.FuncMap))
//
func FuncMap() template.FuncMap {
	return template.FuncMap(genericMap)
}

// TextFuncMap returns a 'text/template'.FuncMap
func TxtFuncMap() ttemplate.FuncMap {
	return ttemplate.FuncMap(genericMap)
}

// HtmlFuncMap returns an 'html/template'.Funcmap
func HtmlFuncMap() template.FuncMap {
	return template.FuncMap(genericMap)
}

var genericMap = map[string]interface{}{
	"hello": func() string { return "Hello!" },

	// Date functions
	"date":         date,
	"date_in_zone": dateInZone,
	"date_modify":  dateModify,
	"now":          func() time.Time { return time.Now() },

	// Strings
	"trim":   strings.TrimSpace,
	"upper":  strings.ToUpper,
	"lower":  strings.ToLower,
	"title":  strings.Title,
	"substr": substring,
	// Switch order so that "foo" | repeat 5
	"repeat": func(count int, str string) string { return strings.Repeat(str, count) },
	// Switch order so that "$foo" | trimall "$"
	"trimall": func(a, b string) string { return strings.Trim(b, a) },

	// Wrap Atoi to stop errors.
	"atoi": func(a string) int { i, _ := strconv.Atoi(a); return i },

	//"gt": func(a, b int) bool {return a > b},
	//"gte": func(a, b int) bool {return a >= b},
	//"lt": func(a, b int) bool {return a < b},
	//"lte": func(a, b int) bool {return a <= b},

	// split "/" foo/bar returns map[int]string{0: foo, 1: bar}
	"split": split,

	// VERY basic arithmetic.
	"add1":    func(i int) int { return i + 1 },
	"add":     func(a, b int) int { return a + b },
	"sub":     func(a, b int) int { return a - b },
	"div":     func(a, b int) int { return a / b },
	"mod":     func(a, b int) int { return a % b },
	"mul":     func(a, b int) int { return a * b },
	"biggest": biggest,

	// string slices. Note that we reverse the order b/c that's better
	// for template processing.
	"join": func(sep string, ss []string) string { return strings.Join(ss, sep) },

	// Defaults
	"default": dfault,

	// Reflection
	"typeOf":     typeOf,
	"typeIs":     typeIs,
	"typeIsLike": typeIsLike,
	"kindOf":     kindOf,
	"kindIs":     kindIs,

	//Slices
	"stringInSlice": stringInSlice,
}

func split(sep, orig string) map[string]string {
	parts := strings.Split(orig, sep)
	res := make(map[string]string, len(parts))
	for i, v := range parts {
		res["_"+strconv.Itoa(i)] = v
	}
	return res
}

// substring creates a substring of the given string.
//
// If start is < 0, this calls string[:length].
//
// If start is >= 0 and length < 0, this calls string[start:]
//
// Otherwise, this calls string[start, length].
func substring(start, length int, s string) string {
	if start < 0 {
		return s[:length]
	}
	if length < 0 {
		return s[start:]
	}
	return s[start:length]
}

// Given a format and a date, format the date string.
//
// Date can be a `time.Time` or an `int, int32, int64`.
// In the later case, it is treated as seconds since UNIX
// epoch.
func date(fmt string, date interface{}) string {
	return dateInZone(fmt, date, "Local")
}

func dateInZone(fmt string, date interface{}, zone string) string {
	var t time.Time
	switch date := date.(type) {
	default:
		t = time.Now()
	case time.Time:
		t = date
	case int64:
		t = time.Unix(date, 0)
	case int:
		t = time.Unix(int64(date), 0)
	case int32:
		t = time.Unix(int64(date), 0)
	}

	loc, err := time.LoadLocation(zone)
	if err != nil {
		loc, _ = time.LoadLocation("UTC")
	}

	return t.In(loc).Format(fmt)
}

func dateModify(fmt string, date time.Time) time.Time {
	d, err := time.ParseDuration(fmt)
	if err != nil {
		return date
	}
	return date.Add(d)
}

func biggest(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// dfault checks whether `given` is set, and returns default if not set.
//
// This returns `d` if `given` appears not to be set, and `given` otherwise.
//
// For numeric types 0 is unset.
// For strings, maps, arrays, and slices, len() = 0 is considered unset.
// For bool, false is unset.
// Structs are never considered unset.
//
// For everything else, including pointers, a nil value is unset.
func dfault(d, given interface{}) interface{} {

	g := reflect.ValueOf(given)
	if !g.IsValid() {
		return d
	}

	set := false

	// Basically adapted from text/template.isTrue
	switch g.Kind() {
	default:
		set = !g.IsNil()
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		set = g.Len() != 0
	case reflect.Bool:
		set = g.Bool()
	case reflect.Complex64, reflect.Complex128:
		set = g.Complex() != 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		set = g.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		set = g.Uint() != 0
	case reflect.Float32, reflect.Float64:
		set = g.Float() != 0
	case reflect.Struct:
		set = true
	}

	if set {
		return given
	}
	return d
}

// typeIs returns true if the src is the type named in target.
func typeIs(target string, src interface{}) bool {
	return target == typeOf(src)
}

func typeIsLike(target string, src interface{}) bool {
	t := typeOf(src)
	return target == t || "*"+target == t
}

func typeOf(src interface{}) string {
	return fmt.Sprintf("%T", src)
}

func kindIs(target string, src interface{}) bool {
	return target == kindOf(src)
}

func kindOf(src interface{}) string {
	return reflect.ValueOf(src).Kind().String()
}

func stringInSlice(a string, list []string) string {
	for _, b := range list {
		if b == a {
			return "t"
		}
	}
	return "f"
}
