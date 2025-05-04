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
	panic(fmt.Sprintf("Unimplemented instruction %q called at PC: 0x%04X", instr.String(), cpu.Registers.PC-1))
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
	value := cpu.Registers.getA()
	cpu.Mmu.WriteByteAt(addr, value)

	log.Printf(
		"0x%04X:\t%-12s ; BC=0x%04X → [0x%04X]=0x%02X",
		cpu.Registers.PC-1,
		instr.String(),
		addr,
		addr,
		value,
	)

	return instr.Cycles[0]
}

func OpLdSPN16(cpu *CPU, instr *Instruction) int {
	value := cpu.fetchInstructionWord()
	cpu.Registers.setSP(value)

	log.Printf(
		"0x%04X:\t%-12s ; nn=0x%04X → SP=0x%04X",
		cpu.Registers.PC-1,
		instr.String(),
		value,
		cpu.Registers.getSP(),
	)

	return instr.Cycles[0]
}

func OpXorAA(cpu *CPU, instr *Instruction) int {
	orig := cpu.Registers.getA()
	result := orig ^ orig
	cpu.Registers.setA(result)

	cpu.Registers.SetFlag(ZeroFlag, true)
	cpu.Registers.SetFlag(SubtractFlag, false)
	cpu.Registers.SetFlag(HalfCarryFlag, false)
	cpu.Registers.SetFlag(CarryFlag, false)

	log.Printf(
		"0x%04X:\t%-12s ; A=0x%02X ^ 0x%02X → A=0x%02X ; Z=%t N=%t H=%t C=%t",
		cpu.Registers.PC-1,
		instr.String(),
		orig,
		orig,
		result,
		true,
		false,
		false,
		false,
	)

	return instr.Cycles[0]
}

func OpLdHLN16(cpu *CPU, instr *Instruction) int {
	value := cpu.fetchInstructionWord()
	cpu.Registers.setHL(value)

	log.Printf(
		"0x%04X:\t%-12s ; nn=0x%04X → HL=0x%04X",
		cpu.Registers.PC-1,
		instr.String(),
		value,
		cpu.Registers.getHL(),
	)

	return instr.Cycles[0]
}

func OpLdMemHLDecA(cpu *CPU, instr *Instruction) int {
	addr := cpu.Registers.getHL()
	value := cpu.Registers.getA()
	cpu.Mmu.WriteByteAt(addr, value)
	cpu.Registers.decHL()

	log.Printf(
		"0x%04X:\t%-12s ; [0x%04X]=0x%02X → A=0x%02X ; HL=0x%04X",
		cpu.Registers.PC-1,
		instr.String(),
		addr,
		value,
		cpu.Registers.getA(),
		cpu.Registers.getHL(),
	)

	return instr.Cycles[0]
}

func OpCbBit7H(cpu *CPU, instr *Instruction) int {
	h := cpu.Registers.getH()
	b7 := (h >> 7) & 1
	zero := b7 == 0
	cpu.Registers.SetFlag(ZeroFlag, zero)
	cpu.Registers.SetFlag(SubtractFlag, false)
	cpu.Registers.SetFlag(HalfCarryFlag, true)

	log.Printf(
		"0x%04X:\t%-12s ; H=0x%02X bit7=%d → Z=%t N=%t H=%t",
		cpu.Registers.PC-1,
		instr.String(),
		h,
		b7,
		zero,
		false,
		true,
	)

	return instr.Cycles[0]
}

func OpJrNZE8(cpu *CPU, instr *Instruction) int {
	offset := cpu.fetchInstructionByte()
	pcBefore := cpu.Registers.getPC() - 1
	if !cpu.Registers.GetFlag(ZeroFlag) {
		cpu.Registers.addPC(int8(offset))
		log.Printf(
			"0x%04X:\t%-12s ; offset=0x%02X → PC=0x%04X (jump taken)",
			pcBefore,
			instr.String(),
			offset,
			cpu.Registers.PC,
		)
		return instr.Cycles[0]
	}
	log.Printf(
		"0x%04X:\t%-12s ; zero flag set → no jump",
		pcBefore,
		instr.String(),
	)

	return instr.Cycles[1]
}
