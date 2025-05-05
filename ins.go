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
	value := cpu.fetchWord()
	cpu.Registers.setBC(value)

	log.Printf(
		"0x%04X:\t%-12s ; nn=0x%04X → BC=0x%04X",
		cpu.opcodeAddr(instr),
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
		cpu.opcodeAddr(instr),
		instr.String(),
		addr,
		addr,
		value,
	)

	return instr.Cycles[0]
}

func OpLdSPN16(cpu *CPU, instr *Instruction) int {
	value := cpu.fetchWord()
	cpu.Registers.setSP(value)

	log.Printf(
		"0x%04X:\t%-12s ; nn=0x%04X → SP=0x%04X",
		cpu.opcodeAddr(instr),
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

	cpu.Registers.setFlag(ZeroFlag, true)
	cpu.Registers.setFlag(SubtractFlag, false)
	cpu.Registers.setFlag(HalfCarryFlag, false)
	cpu.Registers.setFlag(CarryFlag, false)

	log.Printf(
		"0x%04X:\t%-12s ; A=0x%02X ^ 0x%02X → A=0x%02X ; Z=%t N=%t H=%t C=%t",
		cpu.opcodeAddr(instr),
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
	value := cpu.fetchWord()
	cpu.Registers.setHL(value)

	log.Printf(
		"0x%04X:\t%-12s ; nn=0x%04X → HL=0x%04X",
		cpu.opcodeAddr(instr),
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
		cpu.opcodeAddr(instr),
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
	cpu.Registers.setFlag(ZeroFlag, zero)
	cpu.Registers.setFlag(SubtractFlag, false)
	cpu.Registers.setFlag(HalfCarryFlag, true)

	log.Printf(
		"0x%04X:\t%-12s ; H=0x%02X bit7=%d → Z=%t N=%t H=%t",
		cpu.opcodeAddr(instr),
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
	offset := cpu.fetchByte()
	if !cpu.Registers.getFlag(ZeroFlag) {
		cpu.Registers.addPC(int8(offset))
		log.Printf(
			"0x%04X:\t%-12s ; offset=0x%02X → PC=0x%04X (jump taken)",
			cpu.opcodeAddr(instr),
			instr.String(),
			offset,
			cpu.Registers.PC,
		)
		return instr.Cycles[0]
	}
	log.Printf(
		"0x%04X:\t%-12s ; zero flag set → no jump",
		cpu.opcodeAddr(instr),
		instr.String(),
	)

	return instr.Cycles[1]
}

func OpLdCN8(cpu *CPU, instr *Instruction) int {
	value := cpu.fetchByte()
	cpu.Registers.setC(value)

	log.Printf(
		"0x%04X:\t%-12s ; n=0x%02X → C=0x%02X",
		cpu.opcodeAddr(instr),
		instr.String(),
		value,
		cpu.Registers.getC(),
	)

	return instr.Cycles[0]
}

func OpLdAN8(cpu *CPU, instr *Instruction) int {
	value := cpu.fetchByte()
	cpu.Registers.setA(value)

	log.Printf(
		"0x%04X:\t%-12s ; n=0x%02X → A=0x%02X",
		cpu.opcodeAddr(instr),
		instr.String(),
		value,
		cpu.Registers.getA(),
	)

	return instr.Cycles[0]
}

func OpLdhMemCA(cpu *CPU, instr *Instruction) int {
	addr := uint16(0xFF00) | uint16(cpu.Registers.getC())
	value := cpu.Registers.getA()

	cpu.Mmu.WriteByteAt(addr, value)

	log.Printf(
		"0x%04X:\t%-12s ; C=0x%02X → [0x%04X]=0x%02X",
		cpu.opcodeAddr(instr),
		instr.String(),
		cpu.Registers.getC(),
		addr,
		value,
	)

	return instr.Cycles[0]
}

func OpIncC(cpu *CPU, instr *Instruction) int {
	orig := cpu.Registers.getC()
	res := orig + 1
	cpu.Registers.setC(res)

	cpu.Registers.inc8Flags(orig, res)

	log.Printf("0x%04X:\t%-12s ; C:%02X → %02X ; Z=%t H=%t",
		cpu.opcodeAddr(instr), instr, orig, res,
		cpu.Registers.getFlag(ZeroFlag),
		cpu.Registers.getFlag(HalfCarryFlag),
	)

	return instr.Cycles[0]
}

func OpLdMemHLA(cpu *CPU, instr *Instruction) int {
	addr := cpu.Registers.getHL()
	value := cpu.Registers.getA()

	cpu.Mmu.WriteByteAt(addr, value)

	log.Printf(
		"0x%04X:\t%-12s ; A=0x%02X → [0x%04X]=0x%02X",
		cpu.opcodeAddr(instr),
		instr.String(),
		value,
		addr,
		value,
	)

	return instr.Cycles[0]
}

func OpLdhMemA8A(cpu *CPU, instr *Instruction) int {
	offset := cpu.fetchByte()
	addr := uint16(0xFF00) | uint16(offset)
	value := cpu.Registers.getA()

	cpu.Mmu.WriteByteAt(addr, value)

	log.Printf(
		"0x%04X:\t%-12s ; a8=0x%02X → [0x%04X]=0x%02X",
		cpu.opcodeAddr(instr),
		instr.String(),
		offset,
		addr,
		value,
	)

	return instr.Cycles[0]
}

func OpLdDEN16(cpu *CPU, instr *Instruction) int {
	value := cpu.fetchWord()
	cpu.Registers.setDE(value)

	log.Printf(
		"0x%04X:\t%-12s ; nn=0x%04X → DE=0x%04X",
		cpu.opcodeAddr(instr),
		instr.String(),
		value,
		cpu.Registers.getDE(),
	)

	return instr.Cycles[0]
}

func OpLdAMemDE(cpu *CPU, instr *Instruction) int {
	addr := cpu.Registers.getDE()
	value := cpu.Mmu.ReadByteAt(addr)
	cpu.Registers.setA(value)

	log.Printf(
		"0x%04X:\t%-12s ; [0x%04X]=0x%02X → A=0x%02X",
		cpu.opcodeAddr(instr),
		instr.String(),
		addr,
		value,
		cpu.Registers.getA(),
	)

	return instr.Cycles[0]
}

func OpCallA16(cpu *CPU, instr *Instruction) int {
	addr := cpu.fetchWord()
	ret := cpu.Registers.getPC()
	cpu.pushWord(ret)
	cpu.Registers.setPC(addr)

	log.Printf(
		"0x%04X:\t%-12s ; nn=0x%04X → PC=0x%04X ; return=0x%04X pushed ; SP=0x%04X",
		cpu.opcodeAddr(instr),
		instr.String(),
		addr,
		cpu.Registers.getPC(), // now equals addr
		ret,
		cpu.Registers.getSP(),
	)
	return instr.Cycles[0]
}

func OpLdCA(cpu *CPU, instr *Instruction) int {
	value := cpu.Registers.getA()
	cpu.Registers.setC(value)

	log.Printf(
		"0x%04X:\t%-12s ; A=0x%02X → C=0x%02X",
		cpu.opcodeAddr(instr),
		instr.String(),
		value,
		cpu.Registers.getC(),
	)

	return instr.Cycles[0]
}

func OpLdBN8(cpu *CPU, instr *Instruction) int {
	value := cpu.fetchByte()
	cpu.Registers.setB(value)

	log.Printf(
		"0x%04X:\t%-12s ; n=0x%02X → B=0x%02X",
		cpu.opcodeAddr(instr),
		instr.String(),
		value,
		cpu.Registers.getB(),
	)

	return instr.Cycles[0]
}

func OpPushBC(cpu *CPU, instr *Instruction) int {
	value := cpu.Registers.getBC()
	cpu.pushWord(value)

	log.Printf(
		"0x%04X:\t%-12s ; BC=0x%04X → [SP]=0x%04X ; SP=0x%04X",
		cpu.opcodeAddr(instr),
		instr.String(),
		value,
		value,
		cpu.Registers.getSP(),
	)
	return instr.Cycles[0]
}

func OpCbRlC(cpu *CPU, instr *Instruction) int {
	old := cpu.Registers.getC()
	carryIn := byte(0)
	if cpu.Registers.getFlag(CarryFlag) {
		carryIn = 1
	}
	newCarry := (old>>7)&1 == 1
	result := (old<<1)&0xFE | carryIn
	cpu.Registers.setC(result)

	cpu.Registers.setFlag(ZeroFlag, result == 0)
	cpu.Registers.setFlag(SubtractFlag, false)
	cpu.Registers.setFlag(HalfCarryFlag, false)
	cpu.Registers.setFlag(CarryFlag, newCarry)

	log.Printf(
		"0x%04X:\t%-12s ; C=0x%02X → 0x%02X ; Z=%t N=%t H=%t C=%t",
		cpu.opcodeAddr(instr),
		instr.String(),
		old,
		result,
		result == 0,
		false,
		false,
		newCarry,
	)

	return instr.Cycles[0]
}

func OpRla(cpu *CPU, instr *Instruction) int {
	a := cpu.Registers.getA()
	carryIn := cpu.Registers.getFlag(CarryFlag)
	newCarry := (a & 0x80) != 0
	res := (a << 1) | boolToUint8(carryIn)

	cpu.Registers.setA(res)

	cpu.Registers.setFlag(ZeroFlag, false)
	cpu.Registers.setFlag(SubtractFlag, false)
	cpu.Registers.setFlag(HalfCarryFlag, false)
	cpu.Registers.setFlag(CarryFlag, newCarry)

	log.Printf(
		"0x%04X:\t%-12s ; A=0x%02X → A=0x%02X ; Z=false N=false H=false C=%t",
		cpu.opcodeAddr(instr),
		instr.String(),
		a,
		res,
		newCarry,
	)

	return instr.Cycles[0]
}

func OpPopBC(cpu *CPU, instr *Instruction) int {
	value := cpu.popWord()
	cpu.Registers.setBC(value)

	log.Printf(
		"0x%04X:\t%-12s ; [SP]=0x%04X → BC=0x%04X ; SP=0x%04X",
		cpu.opcodeAddr(instr),
		instr.String(),
		value,
		cpu.Registers.getBC(),
		cpu.Registers.getSP(),
	)
	return instr.Cycles[0]
}
