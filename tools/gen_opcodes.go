//go:build ignore
// +build ignore

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"log"
	"os"
	"sort"
	"strings"
)

// rawInstr represents the structure directly parsed from the JSON opcode definitions.
type rawInstr struct {
	Opcode     string            `json:"opcode"`
	Mnemonic   string            `json:"mnemonic"`
	Bytes      uint8             `json:"bytes"`
	Cycles     []int             `json:"cycles"`
	CbPrefixed bool              `json:"cbprefixed"`
	Operands   []rawOp           `json:"operands"`
	Immediate  bool              `json:"immediate"`
	Flags      map[string]string `json:"flags"`
}

// rawOp represents the operand structure parsed from JSON.
type rawOp struct {
	Name      string `json:"name"`
	Bytes     uint8  `json:"bytes,omitempty"`
	Immediate bool   `json:"immediate"`
	Increment bool   `json:"increment,omitempty"`
	Decrement bool   `json:"decrement,omitempty"`
}

// jsonInstructions holds the structure of the entire opcodes.json file.
type jsonInstructions struct {
	Unprefixed map[string]rawInstr `json:"unprefixed"`
	Cbprefixed map[string]rawInstr `json:"cbprefixed"`
}

func main() {
	data, err := os.ReadFile("data/opcodes.json")
	if err != nil {
		log.Fatalf("Error reading opcodes.json: %v", err)
	}

	var js jsonInstructions
	if err := json.Unmarshal(data, &js); err != nil {
		log.Fatalf("Error unmarshalling opcodes.json: %v", err)
	}

	if err := writeOpcodes("opcodes_gen.go", js); err != nil {
		log.Fatalf("Error writing opcodes_gen.go: %v", err)
	}

	fmt.Println("Successfully generated opcodes_gen.go")
}

func writeOpcodes(path string, js jsonInstructions) error {
	var buf bytes.Buffer

	// Header
	fmt.Fprintln(&buf, "// Code generated by tools/gen_opcodes.go; DO NOT EDIT.")
	fmt.Fprintln(&buf, "package main")
	fmt.Fprintln(&buf)
	fmt.Fprintln(&buf, "//nolint:lll // Keeping lines long for generated code clarity")
	fmt.Fprintln(&buf, "var opcodes = Instructions{")

	// Generate entries for all 512 slots
	for i := range 512 {
		var entry *rawInstr
		var actualOpcodeValue int
		isCbPrefixed := false
		hexKey := ""

		if i < 256 {
			actualOpcodeValue = i
			hexKey = fmt.Sprintf("0x%02X", actualOpcodeValue)
			if ri, ok := js.Unprefixed[hexKey]; ok {
				entry = &ri
			}
		} else {
			isCbPrefixed = true
			actualOpcodeValue = i - 256
			hexKey = fmt.Sprintf("0x%02X", actualOpcodeValue)
			if ri, ok := js.Cbprefixed[hexKey]; ok {
				entry = &ri
				entry.CbPrefixed = true
			}
		}

		if entry == nil {
			fmt.Fprintf(&buf, "\t{}, // Index %d (Opcode: %s, CB: %v)\n", i, hexKey, isCbPrefixed)
			continue
		}

		// Cycles
		cycleStrs := make([]string, len(entry.Cycles))
		for j, c := range entry.Cycles {
			cycleStrs[j] = fmt.Sprintf("%d", c)
		}
		cyclesLiteral := "[]int{" + strings.Join(cycleStrs, ", ") + "}"
		if len(entry.Cycles) == 0 {
			cyclesLiteral = "nil"
		}

		// Operands
		operandStrings := make([]string, len(entry.Operands))
		for j, op := range entry.Operands {
			parts := []string{fmt.Sprintf(`Name: %q`, op.Name)}
			if op.Bytes > 0 {
				parts = append(parts, fmt.Sprintf("Bytes: %d", op.Bytes))
			}
			if op.Immediate {
				parts = append(parts, "Immediate: true")
			}
			if op.Increment {
				parts = append(parts, "Increment: true")
			}
			if op.Decrement {
				parts = append(parts, "Decrement: true")
			}
			sort.Strings(parts[1:])
			operandStrings[j] = "Operand{" + strings.Join(parts, ", ") + "}"
		}
		operandsLiteral := "[]Operand{" + strings.Join(operandStrings, ", ") + "}"
		if len(entry.Operands) == 0 {
			operandsLiteral = "nil"
		}

		// Flags
		flagKeys := make([]string, 0, len(entry.Flags))
		for k := range entry.Flags {
			flagKeys = append(flagKeys, k)
		}
		sort.Strings(flagKeys)
		flagStrings := make([]string, len(flagKeys))
		for i, k := range flagKeys {
			flagStrings[i] = fmt.Sprintf("%q: %q", k, entry.Flags[k])
		}
		flagsLiteral := "map[string]string{" + strings.Join(flagStrings, ", ") + "}"
		if len(entry.Flags) == 0 {
			flagsLiteral = "nil"
		}

		// Emit the struct literal
		fmt.Fprintf(&buf,
			"\t{Opcode: %q, Mnemonic: %q, Bytes: %d, Cycles: %s, CbPrefixed: %v, Operands: %s, Immediate: %v, Flags: %s},\n",
			hexKey,
			entry.Mnemonic,
			entry.Bytes,
			cyclesLiteral,
			isCbPrefixed,
			operandsLiteral,
			entry.Immediate,
			flagsLiteral,
		)
	}

	fmt.Fprintln(&buf, "}")

	// Run through go/format
	src, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("format.Source error: %w\n\nunformatted source:\n%s", err, buf.String())
	}

	// Write the formatted code out
	if err := os.WriteFile(path, src, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}
