// Copyright ©2020 The go-latex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore
// +build ignore

//
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sort"
)

func main() {
	f, err := os.Create("symbols_gen.go")
	if err != nil {
		log.Fatalf("could not create symbols file: %+v", err)
	}
	defer f.Close()

	gen(f,
		sym{"_binary_operators", "BinaryOperators"},
		sym{"_relation_symbols", "RelationSymbols"},
		sym{"_arrow_symbols", "ArrowSymbols"},
		sym{"_punctuation_symbols", "PunctuationSymbols"},
		sym{"_overunder_symbols", "OverUnderSymbols"},
		sym{"_overunder_functions", "OverUnderFunctions"},
		sym{"_dropsub_symbols", "DropSubSymbols"},
		sym{"_fontnames", "FontNames"},
		sym{"_function_names", "FunctionNames"},
		sym{"_ambi_delim", "AmbiDelim"},
		sym{"_left_delim", "LeftDelim"},
		sym{"_right_delim", "RightDelim"},
	)

	err = f.Close()
	if err != nil {
		log.Fatalf("could not close symbols file: %+v", err)
	}
}

type sym struct {
	Py string `json:"py"`
	Go string `json:"go"`
}

func gen(o io.Writer, syms ...sym) error {
	r := new(bytes.Buffer)
	err := json.NewEncoder(r).Encode(syms)
	if err != nil {
		return fmt.Errorf("could not encode input JSON: %+v", err)
	}

	stdout := new(bytes.Buffer)
	cmd := exec.Command("python", "-c", py, r.String())
	cmd.Stdout = stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("could not run python script: %+v", err)
	}

	var out map[string][]string
	err = json.NewDecoder(stdout).Decode(&out)
	if err != nil {
		return fmt.Errorf("could not decode output JSON: %+v", err)
	}

	fmt.Fprintf(o, `// Autogenerated. DO NOT EDIT.

package symbols

var (
`)

	keys := make([]string, 0, len(out))
	for k := range out {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i := range keys {
		k := keys[i]
		v := out[k]
		if i > 0 {
			fmt.Fprintf(o, "\n")
		}
		fmt.Fprintf(o, "\t%s = NewSet(\n", k)
		for _, sym := range v {
			fmt.Fprintf(o, "\t\t%q,\n", sym)
		}
		fmt.Fprintf(o, "\t)\n")
	}

	fmt.Fprintf(o, ")\n")
	return nil
}

const py = `
import sys
import string
import json
import matplotlib.mathtext as mtex

input = json.loads(sys.argv[1])
data = {}
for v in input:
	symbols = getattr(mtex.Parser, v["py"])
	data[v["go"]] = list(symbols)

json.dump(data, sys.stdout)
sys.stdout.flush()
`
