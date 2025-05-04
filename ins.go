package main

import (
	"fmt"
	"strings"
)

type Instructions [512]Instruction

type Instruction struct {
	Opcode     string                       `json:"opcode"`
	Mnemonic   string                       `json:"mnemonic"`
	Bytes      uint8                        `json:"bytes"`
	Cycles     []int                        `json:"cycles"`
	Execute    func(*CPU, *Instruction) int `json:"-"`
	CbPrefixed bool                         `json:"cbprefixed"`
	Operands   []Operand                    `json:"operands"`
	Immediate  bool                         `json:"immediate"`
	Flags      map[string]string            `json:"flags"`
}

type Operand struct {
	Name      string `json:"name"`
	Bytes     uint8  `json:"bytes,omitempty"`
	Immediate bool   `json:"immediate"`
	Increment bool   `json:"increment,omitempty"`
	Decrement bool   `json:"decrement,omitempty"`
}

// A human-readable representation of the instruction
func (i *Instruction) String() string {
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
		if !operand.Immediate {
			operandStr = "(" + operandStr + ")"
		}

		operands[idx] = operandStr
	}
	operandsStr := strings.Join(operands, ", ")

	return fmt.Sprintf("%s %s", i.Mnemonic, operandsStr)
}

var instructionImplementations = map[string]func(*CPU, *Instruction) int{
	"NOP": nop,
	// "LD":  ld,
}

func unimplemented(cpu *CPU, instr *Instruction) int {
	panic(fmt.Sprintf("Unimplemented instruction called at PC: 0x%04X", cpu.Registers.PC-1))
}

func nop(cpu *CPU, instr *Instruction) int {
	// TODO
	return 1
}

// func ld(cpu *CPU, instr *Instruction) int {
// 	target := instr.Operands[0]
// 	source := instr.Operands[1]

// 	var sourceVal uint16
// 	switch source.Name {
// 	case "n16":
// 		sourceVal = cpu.fetchInstructionWord()
// 	case "A":
// 		sourceVal = cpu.Registers.getA()
// 	}

// 	switch target.Name {
// 	case "BC":
// 		cpu.Registers.setBC(sourceVal)
// 	}
// }
