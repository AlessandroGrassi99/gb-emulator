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
	"unicode"
)

// rawInstr mirrors the JSON structure in data/opcodes.json
type rawInstr struct {
	Mnemonic string  `json:"mnemonic"`
	Bytes    uint8   `json:"bytes"`
	Operands []rawOp `json:"operands"`
	// Add other fields if needed for analysis, but Mnemonic and Operands are key
}

// rawOp represents operand details from JSON.
type rawOp struct {
	Name      string `json:"name"`
	Bytes     uint8  `json:"bytes,omitempty"`
	Immediate bool   `json:"immediate"` // Note: This flag's meaning in JSON can be ambiguous
	Increment bool   `json:"increment,omitempty"`
	Decrement bool   `json:"decrement,omitempty"`
}

// jsonInstructions holds unprefixed and CB-prefixed instruction sets.
type jsonInstructions struct {
	Unprefixed map[string]rawInstr `json:"unprefixed"`
	Cbprefixed map[string]rawInstr `json:"cbprefixed"`
}

// DispatchEntry holds the generated function name and the original human-readable form.
type DispatchEntry struct {
	FuncName            string
	HumanRepresentation string
}

// --- Operand Type Classification Helpers (Revised) ---

func isR8(op rawOp) bool {
	// 8-bit registers (A, B, C, D, E, H, L)
	switch op.Name {
	case "A", "B", "C", "D", "E", "H", "L":
		return true
	default:
		return false
	}
}

func isR16(op rawOp) bool {
	// 16-bit register pairs (BC, DE, HL, SP)
	switch op.Name {
	case "BC", "DE", "HL", "SP":
		return true
	default:
		return false
	}
}

func isAF(op rawOp) bool {
	// AF register pair (special case for PUSH/POP)
	return op.Name == "AF"
}

func isN8(op rawOp) bool {
	// Immediate 8-bit value
	return op.Bytes == 1 && op.Name == "n8"
}

func isN16(op rawOp) bool {
	// Immediate 16-bit value
	return op.Bytes == 2 && op.Name == "n16"
}

func isA8(op rawOp) bool {
	// 8-bit address offset (for LDH)
	return op.Bytes == 1 && op.Name == "a8"
}

func isA16(op rawOp) bool {
	// 16-bit immediate address
	return op.Bytes == 2 && op.Name == "a16"
}

func isE8(op rawOp) bool {
	// Signed 8-bit offset (for JR, ADD SP)
	return op.Bytes == 1 && op.Name == "e8"
}

// --- Memory Access Classification Helpers ---

// Is the operand representing memory address via register pair BC? (e.g., LD A, (BC))
// We need to distinguish this from LD BC, n16. Check the Immediate flag from JSON.
// If Immediate is false, it's likely the address mode (BC).
func isMemBC(op rawOp) bool { return op.Name == "BC" && !op.Immediate }
func isMemDE(op rawOp) bool { return op.Name == "DE" && !op.Immediate }
func isMemHL(op rawOp) bool { return op.Name == "HL" && !op.Immediate }

// Is the operand representing memory address via register C? (e.g., LDH A, (C))
func isMemC(op rawOp) bool { return op.Name == "C" && !op.Immediate }

// Is the operand representing memory address (HL) with post-increment/decrement?
func isMemHLIncDec(op rawOp) bool {
	return op.Name == "HL" && !op.Immediate && (op.Increment || op.Decrement)
}

// --- Other Operand Type Helpers ---

// Is the operand a condition code?
func isCond(op rawOp) bool {
	switch op.Name {
	case "Z", "NZ", "C", "NC":
		return true
	default:
		return false
	}
}

// Is the operand a bit index (0-7)?
func isBitIdx(op rawOp) bool {
	return len(op.Name) == 1 && op.Name >= "0" && op.Name <= "7"
}

// Is the operand an RST vector target?
func isRstVec(op rawOp) bool {
	return strings.HasPrefix(op.Name, "$")
}

// --- Main Logic ---

func main() {
	js, err := loadInstructions("data/opcodes.json")
	if err != nil {
		log.Fatalf("Error loading instructions: %v", err)
	}

	dispatchEntries := buildDispatchMap(js)

	if err := writeDispatchMap("opcodes_dispatch_gen.go", dispatchEntries); err != nil {
		log.Fatalf("Error writing opcodes_dispatch_gen.go: %v", err)
	}

	fmt.Println("Successfully generated opcodes_dispatch_gen.go")
}

// loadInstructions reads and parses the JSON instructions file.
func loadInstructions(path string) (*jsonInstructions, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	var js jsonInstructions
	decoder := json.NewDecoder(bytes.NewReader(data))
	// decoder.DisallowUnknownFields() // Uncomment to be strict
	if err := decoder.Decode(&js); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}
	return &js, nil
}

// buildDispatchMap processes raw and CB-prefixed opcodes into a map of exec function keys.
func buildDispatchMap(js *jsonInstructions) map[int]DispatchEntry {
	dispatchEntries := make(map[int]DispatchEntry, 512)

	processOpcodes(js.Unprefixed, "Op", 0, dispatchEntries)
	processOpcodes(js.Cbprefixed, "OpCb", 256, dispatchEntries)

	return dispatchEntries
}

// processOpcodes iterates a set of opcodes, generating keys and populating dispatchMap.
func processOpcodes(set map[string]rawInstr, prefix string, offset int, dispatchMap map[int]DispatchEntry) {
	for hexKey, instr := range set {
		var idxRaw int
		if _, err := fmt.Sscanf(hexKey, "0x%X", &idxRaw); err != nil {
			log.Fatalf("Invalid hex key %q: %v", hexKey, err)
		}
		idx := offset + idxRaw

		if instr.Mnemonic == "" || strings.HasPrefix(instr.Mnemonic, "ILLEGAL") {
			continue
		}

		funcName := determineGroupedFuncName(instr, prefix) // *** Core change here ***
		if funcName == "" {
			// Log instructions that didn't get a grouped name, except known skippable ones
			if instr.Mnemonic != "PREFIX" { // Don't warn about the CB prefix itself
				log.Printf("Warning: No grouped function name determined for %s %s (Opcode: %s)", prefix, instr.String(), hexKey)
			}
			continue // Skip if no mapping decided (or handle as unimplemented)
		}

		entry := DispatchEntry{
			FuncName:            funcName,
			HumanRepresentation: instr.String(), // Keep original representation for comments
		}
		dispatchMap[idx] = entry
	}
}

// determineGroupedFuncName analyzes the instruction and returns a generic function name.
func determineGroupedFuncName(instr rawInstr, prefix string) string {
	m := instr.Mnemonic
	ops := instr.Operands
	opCount := len(ops)

	// --- Simple No-Operand Instructions ---
	switch m {
	case "NOP":
		return prefix + "Nop"
	case "HALT":
		return prefix + "Halt"
	case "STOP":
		return prefix + "Stop" // Note: STOP has a dummy n8 operand in JSON, but treat as no-op func sig here
	case "DI":
		return prefix + "Di"
	case "EI":
		return prefix + "Ei"
	case "RLCA":
		return prefix + "Rlca" // Special Accumulator Rotates
	case "RRCA":
		return prefix + "Rrca"
	case "RLA":
		return prefix + "Rla"
	case "RRA":
		return prefix + "Rra"
	case "DAA":
		return prefix + "Daa"
	case "CPL":
		return prefix + "Cpl"
	case "SCF":
		return prefix + "Scf"
	case "CCF":
		return prefix + "Ccf"
	case "RET": // Handle RET and RET cond
		if opCount == 0 {
			return prefix + "Ret"
		}
		if opCount == 1 && isCond(ops[0]) {
			return prefix + "RetCond"
		}
	case "RETI":
		return prefix + "Reti"
	case "PREFIX":
		return "" // CB Prefix itself doesn't have an Op func
	}

	// --- Instructions by Mnemonic and Operand Pattern ---
	baseName := prefix + goifyMnemonic(m) // e.g., "OpLd", "OpAdd", "OpCbBit"

	switch m {
	// --- Load Instructions (LD, LDH) ---
	case "LD":
		if opCount == 2 {
			op1, op2 := ops[0], ops[1]
			// LD r, r'
			if isR8(op1) && isR8(op2) {
				return baseName + "R8R8"
			}
			// LD r, n8
			if isR8(op1) && isN8(op2) {
				return baseName + "R8N8"
			}
			// LD rr, n16
			if isR16(op1) && isN16(op2) {
				return baseName + "R16N16"
			}
			// LD (BC), A
			if isMemBC(op1) && op2.Name == "A" {
				return baseName + "MemBCA"
			} // Keep specific common forms if desired
			// LD (DE), A
			if isMemDE(op1) && op2.Name == "A" {
				return baseName + "MemDEA"
			}
			// LD (HL), r
			if isMemHL(op1) && isR8(op2) {
				return baseName + "MemHLR8"
			}
			// LD (HL), n8
			if isMemHL(op1) && isN8(op2) {
				return baseName + "MemHLN8"
			}
			// LD A, (BC)
			if op1.Name == "A" && isMemBC(op2) {
				return baseName + "AMemBC"
			} // Keep specific common forms if desired
			// LD A, (DE)
			if op1.Name == "A" && isMemDE(op2) {
				return baseName + "AMemDE"
			}
			// LD r, (HL)
			if isR8(op1) && isMemHL(op2) {
				return baseName + "R8MemHL"
			}
			// LD (a16), A
			if isA16(op1) && op2.Name == "A" {
				return baseName + "MemImm16A"
			}
			// LD (a16), SP
			if isA16(op1) && op2.Name == "SP" {
				return baseName + "MemImm16SP"
			}
			// LD A, (a16)
			if op1.Name == "A" && isA16(op2) {
				return baseName + "AMemImm16"
			}
			// LD SP, HL
			if op1.Name == "SP" && op2.Name == "HL" {
				return baseName + "SPHL"
			} // Specific
			// LD (HL-), A -> LD (HLD), A
			if isMemHL(op1) && op1.Decrement && op2.Name == "A" {
				return baseName + "MemHLDA"
			}
			// LD (HL+), A -> LD (HLI), A
			if isMemHL(op1) && op1.Increment && op2.Name == "A" {
				return baseName + "MemHLIA"
			}
			// LD A, (HL-) -> LD A, (HLD)
			if op1.Name == "A" && isMemHL(op2) && op2.Decrement {
				return baseName + "AMemHLD"
			}
			// LD A, (HL+) -> LD A, (HLI)
			if op1.Name == "A" && isMemHL(op2) && op2.Increment {
				return baseName + "AMemHLI"
			}
		}
		if opCount == 3 { // LD HL, SP+e8
			op1, op2, op3 := ops[0], ops[1], ops[2]
			// Check names and use isE8 helper for the offset
			if op1.Name == "HL" && op2.Name == "SP" && isE8(op3) {
				return baseName + "HLSPImm8"
			}
		}
	case "LDH": // High memory loads
		if opCount == 2 {
			op1, op2 := ops[0], ops[1]
			// LDH (a8), A
			if isA8(op1) && op2.Name == "A" {
				return baseName + "MemImm8A"
			}
			// LDH A, (a8)
			if op1.Name == "A" && isA8(op2) {
				return baseName + "AMemImm8"
			}
			// LDH (C), A
			if isMemC(op1) && op2.Name == "A" {
				return baseName + "MemCA"
			}
			// LDH A, (C)
			if op1.Name == "A" && isMemC(op2) {
				return baseName + "AMemC"
			}
		}

	// --- Arithmetic/Logic (A operand implicit/first) ---
	case "ADD", "ADC", "SUB", "SBC", "AND", "XOR", "OR", "CP":
		if opCount == 2 { // Standard ALU ops are A, X or HL, rr or SP, e8
			op1, op2 := ops[0], ops[1]
			// ALU A, X variants
			if op1.Name == "A" {
				if isR8(op2) {
					return baseName + "AR8"
				}
				if isN8(op2) {
					return baseName + "AN8"
				}
				if isMemHL(op2) {
					return baseName + "AMemHL"
				}
			}
			// ADD HL, rr
			if m == "ADD" && op1.Name == "HL" && isR16(op2) {
				return baseName + "HLR16"
			}
			// ADD SP, e8
			if m == "ADD" && op1.Name == "SP" && isE8(op2) {
				return baseName + "SPE8"
			}
		}

	// --- Increment/Decrement ---
	case "INC", "DEC":
		if opCount == 1 {
			op1 := ops[0]
			if isR8(op1) {
				return baseName + "R8"
			}
			if isR16(op1) {
				return baseName + "R16"
			}
			if isMemHL(op1) {
				return baseName + "MemHL"
			}
		}

	// --- Rotates/Shifts (CB prefixed or direct A) ---
	// Note: Direct A rotates (RLCA, etc.) handled in simple cases above
	case "RLC", "RRC", "RL", "RR", "SLA", "SRA", "SWAP", "SRL":
		if opCount == 1 {
			op1 := ops[0]
			if isR8(op1) {
				return baseName + "R8"
			} // CB: OpCbSlaR8, etc.
			if isMemHL(op1) {
				return baseName + "MemHL"
			} // CB: OpCbSlaMemHL, etc.
		}

	// --- Bit Operations (CB prefixed) ---
	case "BIT", "RES", "SET":
		if opCount == 2 {
			op1, op2 := ops[0], ops[1]
			if isBitIdx(op1) { // First operand is the bit index
				if isR8(op2) {
					return baseName + "BR8"
				} // CB: OpCbBitBR8, etc.
				if isMemHL(op2) {
					return baseName + "BMemHL"
				} // CB: OpCbBitBMemHL, etc.
			}
		}

	// --- Jumps / Calls ---
	case "JP":
		if opCount == 1 {
			op1 := ops[0]
			if isA16(op1) {
				return baseName + "Imm16"
			} // JP a16
			// JP HL uses immediate=true in JSON, but is a register load
			if op1.Name == "HL" {
				return baseName + "HL"
			}
		}
		if opCount == 2 { // JP cond, a16
			op1, op2 := ops[0], ops[1]
			if isCond(op1) && isA16(op2) {
				return baseName + "CondImm16"
			}
		}
	case "JR":
		if opCount == 1 { // JR e8
			op1 := ops[0]
			if isE8(op1) {
				return baseName + "Imm8"
			}
		}
		if opCount == 2 { // JR cond, e8
			op1, op2 := ops[0], ops[1]
			if isCond(op1) && isE8(op2) {
				return baseName + "CondImm8"
			}
		}
	case "CALL":
		if opCount == 1 { // CALL a16
			op1 := ops[0]
			if isA16(op1) {
				return baseName + "Imm16"
			}
		}
		if opCount == 2 { // CALL cond, a16
			op1, op2 := ops[0], ops[1]
			if isCond(op1) && isA16(op2) {
				return baseName + "CondImm16"
			}
		}

	// --- Returns ---
	// RET, RETI handled as simple cases above

	// --- Stack ---
	case "PUSH", "POP":
		if opCount == 1 {
			op1 := ops[0]
			// Group AF with other R16 for PUSH/POP
			if isR16(op1) || isAF(op1) {
				return baseName + "R16"
			}
		}

	// --- Restarts ---
	case "RST":
		if opCount == 1 && isRstVec(ops[0]) {
			return baseName + "Vec"
		}
	}

	// If we reach here, the instruction pattern wasn't matched by the logic above
	return "" // Indicate no grouped name found
}

// writeDispatchMap generates the Go source mapping opcodes to exec functions.
func writeDispatchMap(path string, dispatchMap map[int]DispatchEntry) error {
	var buf bytes.Buffer

	fmt.Fprintln(&buf, "// Code generated by tools/gen_opcodes_dispatch.go; DO NOT EDIT.")
	fmt.Fprintln(&buf, "package main")
	fmt.Fprintln(&buf)
	fmt.Fprintln(&buf, "//nolint:lll // Keeping lines long for generated code clarity")
	fmt.Fprintln(&buf, "var opcodesFunc = map[int]func(*CPU,*Instruction)int{")

	indices := make([]int, 0, len(dispatchMap))
	for i := range dispatchMap {
		indices = append(indices, i)
	}
	sort.Ints(indices)

	// Track function names already used to avoid redundant comments for the same group func
	// usedFuncNames := make(map[string]bool) // Optional: Can be used to simplify comments

	for _, i := range indices {
		entry := dispatchMap[i]
		if entry.FuncName == "" {
			continue
		}
		desc := entry.HumanRepresentation
		comment := fmt.Sprintf("// %s", desc)
		// Optional: Simplify comment if func name already seen
		// if usedFuncNames[entry.FuncName] {
		//    comment = fmt.Sprintf("// -> %s", entry.FuncName)
		// } else {
		//    usedFuncNames[entry.FuncName] = true
		// }

		if i < 256 {
			fmt.Fprintf(&buf, "\t0x%02X: %s, %s\n", i, entry.FuncName, comment)
		} else {
			fmt.Fprintf(&buf, "\t%d /* CB 0x%02X */: %s, %s\n",
				i, i-256, entry.FuncName, comment)
		}
	}

	fmt.Fprintln(&buf, "}")

	src, err := format.Source(buf.Bytes())
	if err != nil {
		// Output the unformatted source for easier debugging
		log.Printf("Error during go/format:\n%s\n", buf.String())
		return fmt.Errorf("go/format error: %w", err)
	}

	return os.WriteFile(path, src, 0644)
}

// goifyMnemonic converts "LD" to "Ld", "ADD" to "Add" etc.
func goifyMnemonic(mn string) string {
	if mn == "" {
		return ""
	}
	runes := []rune(strings.ToLower(mn))
	if len(runes) > 0 {
		runes[0] = unicode.ToUpper(runes[0])
	}
	return string(runes)
}

// --- Helper for String Representation (Revised) ---
func (i *rawInstr) String() string {
	if i.Mnemonic == "" {
		return "UNKNOWN"
	}
	if len(i.Operands) == 0 {
		return i.Mnemonic
	}

	operands := make([]string, len(i.Operands))
	for idx, operand := range i.Operands {
		operandStr := operand.Name
		if operand.Increment {
			operandStr = operandStr + "++"
		}
		if operand.Decrement {
			operandStr = operandStr + "--"
		}

		// Determine if wrapping in parentheses is needed based on typical assembly syntax
		// Wrap register pairs used as addresses, C used as address, a8/a16 addresses
		wrap := false
		// Check if the operand *name* suggests it's used as a memory address pointer
		switch operand.Name {
		case "BC", "DE", "HL", "C": // Register pairs or C register used for addressing (LD A, (BC), LDH A, (C), etc.)
			// Only wrap if the JSON immediate flag is false, indicating it's not the register itself (like in LD BC, n16)
			if !operand.Immediate {
				wrap = true
			}
		case "a8", "a16": // These *always* represent addresses in syntax like (a8), (a16)
			wrap = true
		}

		if wrap {
			operandStr = "(" + operandStr + ")"
		}

		operands[idx] = operandStr
	}
	operandsStr := strings.Join(operands, ", ")

	return fmt.Sprintf("%s %s", i.Mnemonic, operandsStr)
}
