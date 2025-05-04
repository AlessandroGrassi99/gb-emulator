package main

import (
	"fmt"
	"log"
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

func OpUnimplemented(cpu *CPU, instr *Instruction) int {
	panic(fmt.Sprintf("Unimplemented instruction called at PC: 0x%04X", cpu.Registers.PC-1))
}

func OpNop(cpu *CPU, instr *Instruction) int {
	return instr.Cycles[0]
}

func OpLdBCN16(cpu *CPU, instr *Instruction) int {
	value := cpu.fetchInstructionWord()
	cpu.Registers.setBC(value)

	log.Printf(
		"0x%04X:\t%-12s ; nn=0x%04X → BC=0x%04X",
		cpu.Registers.PC-1,
		instr.String(),
		value,
		cpu.Registers.getBC(),
	)

	return instr.Cycles[0]
}

func OpLdMemBCA(cpu *CPU, instr *Instruction) int {
	addr := cpu.Registers.getBC()
	val := cpu.Registers.getA()
	cpu.Mmu.WriteByteAt(addr, val)

	log.Printf(
		"0x%04X:\t%-12s ; BC=0x%04X → [0x%04X]=0x%02X",
		cpu.Registers.PC-1,
		instr.String(),
		addr,
		addr,
		val,
	)

	return instr.Cycles[0]
}
