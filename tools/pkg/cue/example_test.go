// Package cue provides a set of functions for loading and manipulating CUE.
package cue

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

var src = `
{
	a: {
		b: 1
		c: 2
	}
	d: [
		{
			e: 3
		}
	]
}
`

func Example() {
	ctx := cuecontext.New()
	v, err := Compile(ctx, []byte(src))
	if err != nil {
		panic(err)
	}

	if err := Validate(v, cue.Concrete(true)); err != nil {
		panic(err)
	}

	// Replace a.b with 2
	v, err = Replace(ctx, v, "a.b", ctx.CompileString("2"))
	fmt.Printf("a.b: %v\n", v.LookupPath(cue.ParsePath("a.b")))

	// Replace d[0].e with 4
	v, err = Replace(ctx, v, "d[0].e", ctx.CompileString("4"))
	fmt.Printf("d[0].e: %v\n", v.LookupPath(cue.ParsePath("d[0].e")))

	// Delete a.c
	v, err = Delete(ctx, v, "a.c")
	fmt.Printf("a: %v\n", v.LookupPath(cue.ParsePath("a")))

	// output:
	// a.b: 2
	// d[0].e: 4
	// a: {
	// 	b: 2
	// }
}
