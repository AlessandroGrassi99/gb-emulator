package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

type JSONInstructions struct {
	Unprefixed map[string]Instruction `json:"unprefixed"`
	Cbprefixed map[string]Instruction `json:"cbprefixed"`
}

type Instruction struct {
	Opcode     string         `json:"opcode"`
	Name       string         `json:"mnemonic"`
	Bytes      uint8          `json:"bytes"`
	Cycles     []int          `json:"cycles"` // Cycles can have multiple values (e.g., for conditional jumps)
	Execute    func(*CPU) int `json:"-"`
	CbPrefixed bool           `json:"cbprefixed"`
	Operands   []Operand      `json:"operands"`
	Immediate  bool           `json:"immediate"`
}

type Operand struct {
	Name      string `json:"name"`
	Bytes     uint   `json:"bytes,omitzero"`
	Immediate bool   `json:"immediate"`
	Increment bool   `json:"increment,omitzero"`
	Decrement bool   `json:"decrement,omitzero"`
}

type Instructions [512]Instruction

func (i *Instructions) UnmarshalJSON(data []byte) error {
	var raw JSONInstructions

	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("error unmarshaling into raw struct: %w", err)
	}

	for opcodeStr, ins := range raw.Unprefixed {
		hex, err := parseHexToUint8(opcodeStr)
		if err != nil {
			return fmt.Errorf("error processing unprefixed key %s: %w", opcodeStr, err)
		}
		ins.Opcode = opcodeStr
		i[hex] = ins
	}
	for opcodeStr, ins := range raw.Cbprefixed {
		hex, err := parseHexToUint8(opcodeStr)
		if err != nil {
			return fmt.Errorf("error processing cbprefixed key %s: %w", opcodeStr, err)
		}
		ins.Opcode = opcodeStr
		ins.CbPrefixed = true
		i[256+int(hex)] = ins
	}

	for idx := range i {
		instr := &(*i)[idx]
		if instr.Name == "" {
			continue
		}

		executeFn, ok := instructionImplementations[instr.String()]
		if ok {
			instr.Execute = executeFn
		} else {
			instr.Execute = unimplemented
		}
	}

	return nil
}

func (i *Instruction) String() string {
	if len(i.Operands) == 0 {
		return fmt.Sprintf("%s", i.Name)
	}

	operands := make([]string, 0)
	for _, operand := range i.Operands {
		operandStr := operand.Name
		if operand.Increment {
			operandStr = operandStr + "++"
		}
		if operand.Decrement {
			operandStr = operandStr + "--"
		}
		if !operand.Immediate {
			operandStr = "(" + operandStr + ")"
		}

		operands = append(operands, operandStr)
	}
	operandsStr := strings.Join(operands, ", ")

	return fmt.Sprintf("%s %s", i.Name, operandsStr)
}

var instructionImplementations = map[string]func(*CPU) int{
	"NOP": nop,
}

func unimplemented(cpu *CPU) int {
	panic(fmt.Sprintf("Unimplemented instruction called at PC: 0x%04X", cpu.Registers.PC-1))
}

func nop(cpu *CPU) int {
	panic(fmt.Sprintf("Generic LD handler called for opcode 0x%02X", 0))
}
