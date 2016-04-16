package index

import (
	"math/rand"
	"testing"
)

// Example (Macbook Pro, 2.3 GHz Intel Core i7, 16 GB 1600 MHz DDR3):
//
// BenchmarkRandomSmall             5000000           267 ns/op          31 B/op          0 allocs/op
// BenchmarkIndexRemove1000Deps        5000        250111 ns/op           0 B/op          0 allocs/op
//
// This file contains a few examples of benchmarks of the index
// implementation. This currently amounts to benchmarking Go's map type, which
// isn't very interesting. However, if we decided to replace the Index
// implementation, we would want benchmarks for comparison.
//
// Each benchmark seeds math/rand with a const for determinism.

// Perform random manipulations of a small number of packages with a small
// number of dependencies.
func BenchmarkRandomSmall(b *testing.B) {
	rand.Seed(1)
	i := NewIndex()
	ops := []func(){
		func() { i.Index("A", map[string]struct{}{"B": struct{}{}}) },
		func() { i.Index("B", nil) },
		func() { i.Query("A") },
		func() { i.Query("B") },
		func() { i.Remove("A") },
		func() { i.Remove("B") },
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		// TODO: does the overhead of the anonymous func call matter?
		ops[rand.Intn(len(ops))]()
	}
}

// Index and Remove a package with a large number of dependencies.
func BenchmarkIndexRemove1000Deps(b *testing.B) {
	rand.Seed(1)
	i := NewIndex()
	pkgs := genPkgSet(b.N)
	deps := genPkgSet(1000)
	for d := range deps {
		i.Index(d, nil)
	}
	b.ResetTimer()
	for p := range pkgs {
		i.Index(p, deps)
		i.Remove(p)
	}
}

// The functions below are for generating test data and should not be called
// under the benchmark timer.

var scratch = make([]byte, 32)

// genPkg generates a random package name. It is not thread safe.
func genPkg() string {
	rand.Read(scratch)
	return string(scratch)
}

// genPkgSet generates a random set of package names of len n. It is not
// thread safe.
func genPkgSet(n int) map[string]struct{} {
	deps := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		p := genPkg()
		if _, ok := deps[p]; ok {
			panic("getPkg duplicate output")
		}
		deps[p] = struct{}{}
	}
	return deps
}
