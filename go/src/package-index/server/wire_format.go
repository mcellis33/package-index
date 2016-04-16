package server

import "errors"

type Message struct {
	Command      string
	Package      string
	Dependencies map[string]struct{}
}

var (
	errMustEndInNewline = errors.New("must end in newline")
	errTooFewPipes      = errors.New("too few pipes")
	errCommaInPackage   = errors.New("package names may not include the reserved character ','")
	errPipeInPackage    = errors.New("package names may not include the reserved character '|'")
	errEmptyPackage     = errors.New("package name may not be empty string")
)

// parseMessage gets the command, package, and dependencies from a message.
//
// The characters '|', ',', and '\n' are reserved by the message format, and
// the spec does not call out any escaping for them, so package names cannot
// contain those characters. There is no difference in the encoding of
// Dependencies = nil and Dependencies = []string{""}, so "" cannot be a valid
// package name. (Plus, I can't see a reason to name a package "".)
//
// Perf vs readability: we could simplify by using utilities such as
// bytes.Split, but that approach would require extra allocations and copies.
// This is in the fast path and it is a write-once kind of component, so it is
// worthwhile to optimize. If this ended up being _very_ perf-critical we
// could generate it or write it in assembly, but that's obviously overkill
// given the index implementation.
func parseMessage(b []byte) (m Message, err error) {
	if len(b) < 1 {
		err = errMustEndInNewline
		return
	}
	if b[len(b)-1] != '\n' {
		err = errMustEndInNewline
		return
	}

	i := 0

	for {
		if b[i] == '\n' {
			err = errTooFewPipes
			return
		}
		if b[i] == '|' {
			break
		}
		i++
	}
	firstPipe := i
	m.Command = string(b[:firstPipe])
	i++

	for {
		if b[i] == '\n' {
			err = errTooFewPipes
			return
		}
		if b[i] == ',' {
			err = errCommaInPackage
			return
		}
		if b[i] == '|' {
			break
		}
		i++
	}
	secondPipe := i
	if firstPipe+1 == secondPipe {
		err = errEmptyPackage
		return
	}
	m.Package = string(b[firstPipe+1 : secondPipe])
	i++

	if b[i] == '\n' {
		return
	}
	numCommas := 0
	for {
		if b[i] == '|' {
			err = errPipeInPackage
			return
		}
		if b[i] == '\n' {
			break
		}
		if b[i] == ',' {
			numCommas++
		}
		i++
	}
	m.Dependencies = make(map[string]struct{}, numCommas+1)

	i = secondPipe + 1
	depStart := i
	d := 0
	for {
		if b[i] == '\n' {
			if depStart == i {
				err = errEmptyPackage
				return
			}
			m.Dependencies[string(b[depStart:i])] = struct{}{}
			return
		}
		if b[i] == ',' {
			if depStart == i {
				err = errEmptyPackage
				return
			}
			m.Dependencies[string(b[depStart:i])] = struct{}{}
			d++
			depStart = i + 1
		}
		i++
	}
}
