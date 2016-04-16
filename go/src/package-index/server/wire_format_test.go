package server

import (
	"reflect"
	"testing"
)

func TestParseMessage(t *testing.T) {
	type testCase struct {
		in  string
		out Message
		err error
	}
	tcs := []testCase{
		{"", Message{}, errMustEndInNewline},
		{"\n", Message{}, errTooFewPipes},
		{"|\n", Message{}, errTooFewPipes},
		{"||\n", Message{}, errEmptyPackage},
		{"|||\n", Message{}, errEmptyPackage},
		{"A||\n", Message{"A", "", nil}, errEmptyPackage},
		{"|A|\n", Message{"", "A", nil}, nil},
		{"|,|\n", Message{}, errCommaInPackage},
		{"||,\n", Message{}, errEmptyPackage},
		{"A|B|\n", Message{"A", "B", nil}, nil},
		{"A|B|,\n", Message{"A", "B", map[string]struct{}{}}, errEmptyPackage},
		{"A|B,|\n", Message{"A", "", nil}, errCommaInPackage},
		{"A|B|C\n", Message{"A", "B", map[string]struct{}{"C": struct{}{}}}, nil},
		{"A|B|C,\n", Message{"A", "B", map[string]struct{}{"C": struct{}{}}}, errEmptyPackage},
		{"A|B|C|\n", Message{"A", "B", nil}, errPipeInPackage},
		{"A|B|C,C\n", Message{"A", "B", map[string]struct{}{"C": struct{}{}}}, nil},
		{"A|B|C,D\n", Message{"A", "B", map[string]struct{}{"C": struct{}{}, "D": struct{}{}}}, nil},
		{"A,B|C|D,E\n", Message{"A,B", "C", map[string]struct{}{"D": struct{}{}, "E": struct{}{}}}, nil},
		{"A|B,C|D,E\n", Message{"A", "", nil}, errCommaInPackage},
		{"A|B|C,D|E,F\n", Message{"A", "B", nil}, errPipeInPackage},
		{"A,B|C,D|E,F\n", Message{"A,B", "", nil}, errCommaInPackage},
		{"A|B|C,D,E,F,G\n", Message{"A", "B", map[string]struct{}{"C": struct{}{}, "D": struct{}{}, "E": struct{}{}, "F": struct{}{}, "G": struct{}{}}}, nil},
		{"aoeu|snth|aoeu,aoeu,snth,aoeu\n", Message{"aoeu", "snth", map[string]struct{}{"aoeu": struct{}{}, "snth": struct{}{}}}, nil},
		{"ŪņЇ|ЌœđЗ|☺ unicode, € rocks ™\n", Message{"ŪņЇ", "ЌœđЗ", map[string]struct{}{"☺ unicode": struct{}{}, " € rocks ™": struct{}{}}}, nil},
	}
	for i, tc := range tcs {
		out, err := parseMessage([]byte(tc.in))
		if !reflect.DeepEqual(err, tc.err) {
			t.Fatalf("test case %v: err %v, expected %v", i, err, tc.err)
		}
		if !reflect.DeepEqual(out, tc.out) {
			t.Fatalf("test case %v: out %v, expected %v", i, out, tc.out)
		}
	}
}

func TestAllBytes(t *testing.T) {
	b := make([]byte, 256)
	for i := range b {
		b[i] = byte(i)
	}
	m, err := parseMessage(b)
	if err == nil {
		t.Fatal(m)
	}
}
