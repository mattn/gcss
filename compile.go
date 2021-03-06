package gcss

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"strings"
)

// cssFilePath creates and returns the CSS file path.
var cssFilePath = func(path string) string {
	ext := filepath.Ext(path)
	return strings.TrimSuffix(path, ext) + extCSS
}

// Compile parses the GCSS file specified by the path parameter,
// generates a CSS file and returns the two channels: the first
// one returns the path of the generated CSS file and the last
// one returns an error when it occurs.
func Compile(path string) (<-chan string, <-chan error) {
	pathc := make(chan string)
	errc := make(chan error)

	go func() {
		data, err := ioutil.ReadFile(path)

		if err != nil {
			errc <- err
			return
		}

		cssPath := cssFilePath(path)

		bc, berrc := CompileBytes(data)

		done, werrc := write(cssPath, bc, berrc)

		select {
		case <-done:
			pathc <- cssPath
		case err := <-werrc:
			errc <- err
			return
		}
	}()

	return pathc, errc
}

// CompileBytes parses the GCSS byte array passed as the s parameter,
// generates a CSS byte array and returns the two channels: the first
// one returns the CSS byte array and the last one returns an error
// when it occurs.
func CompileBytes(b []byte) (<-chan []byte, <-chan error) {
	lines := strings.Split(formatLF(string(b)), lf)

	bc := make(chan []byte, len(lines))
	errc := make(chan error)

	go func() {
		ctx := newContext()

		elemc, pErrc := parse(lines)

		for {
			select {
			case elem, ok := <-elemc:

				if elem != nil {
					elem.SetContext(ctx)
				}

				switch elem.(type) {
				case *mixinDeclaration:
					v := elem.(*mixinDeclaration)
					ctx.mixins[v.name] = v
				case *variable:
					v := elem.(*variable)
					ctx.vars[v.name] = v
				case *atRule, *declaration, *selector:
					bf := new(bytes.Buffer)
					elem.WriteTo(bf)
					bc <- bf.Bytes()
				}

				if !ok {
					close(bc)
					return
				}
			case err := <-pErrc:
				errc <- err
				return
			}
		}
	}()

	return bc, errc
}
