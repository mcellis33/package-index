package index

import "testing"

// go test -cover coverage: 100.0% of statements
func TestIndex(t *testing.T) {
	i := NewIndex()
	if !i.Remove("A") {
		t.Fatal("remove un-indexed pkg failed")
	}
	if i.Query("A") {
		t.Fatal("query un-indexed pkg succeeded")
	}
	if i.Index("A", map[string]struct{}{"B": struct{}{}}) {
		t.Fatal("index pkg with missing deps succeeded")
	}
	if !i.Index("B", nil) {
		t.Fatal("index pkg with nil deps failed")
	}
	if !i.Index("C", map[string]struct{}{}) {
		t.Fatal("index pkg with map[string]struct{}{} deps failed")
	}
	if !i.Index("A", map[string]struct{}{"B": struct{}{}}) {
		t.Fatal("index pkg with deps failed")
	}
	if !i.Index("", map[string]struct{}{"A": struct{}{}, "B": struct{}{}}) {
		t.Fatal("add empty string failed")
	}
	if !i.Index("D", map[string]struct{}{"": struct{}{}, "A": struct{}{}}) {
		t.Fatal("depend on empty string failed")
	}
	if !i.Query("A") {
		t.Fatal("A missing")
	}
	if !i.Query("B") {
		t.Fatal("B missing")
	}
	if !i.Query("C") {
		t.Fatal("C missing")
	}
	if !i.Query("") {
		t.Fatal("empty string missing")
	}
	if !i.Query("D") {
		t.Fatal("D missing")
	}
	// Should succeed but NOT add C to A's deps
	if !i.Index("A", map[string]struct{}{"B": struct{}{}, "C": struct{}{}}) {
		t.Fatal("index already indexed failed")
	}
	if !i.Remove("C") {
		t.Fatal("remove C failed (should have no parents)")
	}
	if i.Remove("B") {
		t.Fatal("remove B succeeded (A and empty string are parents)")
	}
	if !i.Remove("D") {
		t.Fatal("remove D failed")
	}
	if !i.Remove("") {
		t.Fatal("remove empty string failed")
	}
	if !i.Remove("A") {
		t.Fatal("remove A failed")
	}
	if !i.Remove("B") {
		t.Fatal("remove B failed")
	}
	if i.Query("A") {
		t.Fatal("query A after remove succeeded")
	}
	if i.Query("B") {
		t.Fatal("query B after remove succeeded")
	}
	if i.Query("C") {
		t.Fatal("query C after remove succeeded")
	}
	if i.Query("D") {
		t.Fatal("query D after remove succeeded")
	}
	if i.Query("") {
		t.Fatal("query empty string after remove succeeded")
	}
}
