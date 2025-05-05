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
	log.Printf(
		"0x%04X:\t%-12s",
		cpu.opcodeAddr(instr),
		instr.String(),
	)
	return instr.Cycles[0]
}

// OpLdR16N16 Handles LD rr, n16 (BC, DE, HL, SP)
func OpLdR16N16(cpu *CPU, instr *Instruction) int {
	if len(instr.Operands) < 1 {
		panic(fmt.Sprintf("OpLdR16N16 expects at least 1 operand, got 0 for %s", instr.String()))
	}
	targetReg := instr.Operands[0].Name // "BC", "DE", "HL", "SP"

	value := cpu.fetchWord()

	switch targetReg {
	case "BC":
		cpu.Registers.setBC(value)
	case "DE":
		cpu.Registers.setDE(value)
	case "HL":
		cpu.Registers.setHL(value)
	case "SP":
		cpu.Registers.setSP(value)
	default:
		panic(fmt.Sprintf("OpLdR16N16: Unexpected target register %q", targetReg))
	}

	log.Printf(
		"0x%04X:\t%-12s ; nn=0x%04X → %s=0x%04X",
		cpu.opcodeAddr(instr),
		instr.String(),
		value,
		targetReg,
		value,
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

func OpXorAR8(cpu *CPU, instr *Instruction) int {
	if len(instr.Operands) < 2 {
		panic(fmt.Sprintf("OpXorAR8 expects 2 operands, got %d for %s", len(instr.Operands), instr.String()))
	}
	sourceReg := instr.Operands[1].Name
	getter := cpu.Registers.getRegisterGetter8(sourceReg)
	if getter == nil {
		panic(fmt.Sprintf("OpXorAR8: Unexpected source register %q", sourceReg))
	}

	orig := cpu.Registers.getA()
	srcVal := getter()
	result := orig ^ srcVal
	cpu.Registers.setA(result)

	cpu.Registers.setFlag(ZeroFlag, result == 0)
	cpu.Registers.setFlag(SubtractFlag, false)
	cpu.Registers.setFlag(HalfCarryFlag, false)
	cpu.Registers.setFlag(CarryFlag, false)

	log.Printf(
		"0x%04X:\t%-12s ; A=0x%02X ^ %s=0x%02X → A=0x%02X ; Z=%t N=%t H=%t C=%t",
		cpu.opcodeAddr(instr),
		instr.String(),
		orig,
		sourceReg,
		srcVal,
		result,
		result == 0,
		false,
		false,
		false,
	)

	return instr.Cycles[0]
}

// OpLdMemHLR8 Handles LD (HL), [A, B, C, D, E, H, L] with optional increment/decrement
func OpLdMemHLR8(cpu *CPU, instr *Instruction) int {
	if len(instr.Operands) < 2 {
		panic(fmt.Sprintf("OpLdMemHLR8 expects 2 operands, got %d for %s", len(instr.Operands), instr.String()))
	}
	
	// Check for increment/decrement on first operand
	isIncrement := instr.Operands[0].Increment
	isDecrement := instr.Operands[0].Decrement
	sourceReg := instr.Operands[1].Name
	getter := cpu.Registers.getRegisterGetter8(sourceReg)
	if getter == nil {
		panic(fmt.Sprintf("OpLdMemHLR8: Unexpected source register %q", sourceReg))
	}

	addr := cpu.Registers.getHL()
	value := getter()
	cpu.Mmu.WriteByteAt(addr, value)
	
	// Handle increment/decrement after memory operation
	if isIncrement {
		cpu.Registers.setHL(addr + 1)
		log.Printf(
			"0x%04X:\t%-12s ; [HL=0x%04X]=%s=0x%02X, HL++: HL=0x%04X",
			cpu.opcodeAddr(instr),
			instr.String(),
			addr,
			sourceReg,
			value,
			cpu.Registers.getHL(),
		)
	} else if isDecrement {
		cpu.Registers.setHL(addr - 1)
		log.Printf(
			"0x%04X:\t%-12s ; [HL=0x%04X]=%s=0x%02X, HL--: HL=0x%04X",
			cpu.opcodeAddr(instr),
			instr.String(),
			addr,
			sourceReg,
			value,
			cpu.Registers.getHL(),
		)
	} else {
		log.Printf(
			"0x%04X:\t%-12s ; [HL=0x%04X]=%s=0x%02X",
			cpu.opcodeAddr(instr),
			instr.String(),
			addr,
			sourceReg,
			value,
		)
	}

	return instr.Cycles[0]
}

func OpCbBitBR8(cpu *CPU, instr *Instruction) int {
	if len(instr.Operands) < 2 {
		panic(fmt.Sprintf("OpCbBitBR8 expects 2 operands, got %d for %s", len(instr.Operands), instr.String()))
	}
	
	// Extract bit number from operand name (e.g., "7" from "7")
	bitStr := instr.Operands[0].Name
	var bitNum uint
	_, err := fmt.Sscanf(bitStr, "%d", &bitNum)
	if err != nil || bitNum > 7 {
		panic(fmt.Sprintf("OpCbBitBR8: Invalid bit number %q", bitStr))
	}
	
	// Get the register
	regName := instr.Operands[1].Name
	getter := cpu.Registers.getRegisterGetter8(regName)
	if getter == nil {
		panic(fmt.Sprintf("OpCbBitBR8: Unexpected register %q", regName))
	}
	
	value := getter()
	bitVal := (value >> bitNum) & 1
	zero := bitVal == 0
	
	cpu.Registers.setFlag(ZeroFlag, zero)
	cpu.Registers.setFlag(SubtractFlag, false)
	cpu.Registers.setFlag(HalfCarryFlag, true)
	// Carry flag is preserved

	log.Printf(
		"0x%04X:\t%-12s ; %s=0x%02X bit%s=%d → Z=%t N=%t H=%t C=%t",
		cpu.opcodeAddr(instr),
		instr.String(),
		regName,
		value,
		bitStr,
		bitVal,
		zero,
		false,
		true,
		cpu.Registers.getFlag(CarryFlag), // Log the preserved carry flag
	)

	return instr.Cycles[0]
}

// OpJrCondImm8 Handles JR cond, e8
func OpJrCondImm8(cpu *CPU, instr *Instruction) int {
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

// OpLdR8N8 Handles LD [A, B, C, D, E, H, L], n8
func OpLdR8N8(cpu *CPU, instr *Instruction) int {
	if len(instr.Operands) < 1 {
		panic(fmt.Sprintf("OpLdR8N8 expects at least 1 operand, got 0 for %s", instr.String()))
	}
	targetReg := instr.Operands[0].Name // "A", "B", "C", ...
	value := cpu.fetchByte()
	setter := cpu.Registers.getRegisterSetter8(targetReg)
	if setter == nil {
		panic(fmt.Sprintf("OpLdR8N8: Unexpected target register %q", targetReg))
	}
	setter(value)

	log.Printf(
		"0x%04X:\t%-12s ; n=0x%02X → %s=0x%02X",
		cpu.opcodeAddr(instr),
		instr.String(),
		value,
		targetReg,
		value,
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

// OpIncR8 Handles INC r [A, B, C, D, E, H, L]
func OpIncR8(cpu *CPU, instr *Instruction) int {
	if len(instr.Operands) < 1 {
		panic(fmt.Sprintf("OpIncR8 expects 1 operand, got 0 for %s", instr.String()))
	}
	regName := instr.Operands[0].Name
	getter := cpu.Registers.getRegisterGetter8(regName)
	setter := cpu.Registers.getRegisterSetter8(regName)
	if getter == nil || setter == nil {
		panic(fmt.Sprintf("OpIncR8: Unexpected register %q", regName))
	}

	orig := getter()
	res := orig + 1
	setter(res)

	// Set flags: Z, N=0, H
	cpu.Registers.setFlag(ZeroFlag, res == 0)
	cpu.Registers.setFlag(SubtractFlag, false)
	// Half-Carry is set if carry from bit 3 to bit 4 occurred
	cpu.Registers.setFlag(HalfCarryFlag, (orig&0x0F)+1 > 0x0F)
	// Carry flag is not affected

	log.Printf("0x%04X:\t%-12s ; %s=0x%02X → 0x%02X ; Z=%t N=%t H=%t C=%t",
		cpu.opcodeAddr(instr),
		instr.String(),
		regName, orig, res,
		cpu.Registers.getFlag(ZeroFlag),
		cpu.Registers.getFlag(SubtractFlag),
		cpu.Registers.getFlag(HalfCarryFlag),
		cpu.Registers.getFlag(CarryFlag), // Log unchanged Carry flag
	)

	return instr.Cycles[0]
}

// OpLdhMemImm8A Handles LDH (a8), A
func OpLdhMemImm8A(cpu *CPU, instr *Instruction) int {
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

// OpCallImm16 Handles CALL a16
func OpCallImm16(cpu *CPU, instr *Instruction) int {
	addr := cpu.fetchWord()
	retAddr := cpu.Registers.getPC()
	cpu.pushWord(retAddr)
	cpu.Registers.setPC(addr)

	log.Printf(
		"0x%04X:\t%-12s ; nn=0x%04X → PC=0x%04X; pushed ret=0x%04X; SP=0x%04X",
		cpu.opcodeAddr(instr),
		instr.String(),
		addr,
		cpu.Registers.getPC(), // Log new PC
		retAddr,
		cpu.Registers.getSP(), // Log new SP
	)
	return instr.Cycles[0]
}

// OpLdR8R8 Handles LD r, r' (e.g., LD C, A)
func OpLdR8R8(cpu *CPU, instr *Instruction) int {
	if len(instr.Operands) < 2 {
		panic(fmt.Sprintf("OpLdR8R8 expects 2 operands, got %d for %s", len(instr.Operands), instr.String()))
	}
	targetReg := instr.Operands[0].Name
	sourceReg := instr.Operands[1].Name

	getter := cpu.Registers.getRegisterGetter8(sourceReg)
	setter := cpu.Registers.getRegisterSetter8(targetReg)

	if getter == nil {
		panic(fmt.Sprintf("OpLdR8R8: Unexpected source register %q", sourceReg))
	}
	if setter == nil {
		panic(fmt.Sprintf("OpLdR8R8: Unexpected target register %q", targetReg))
	}

	value := getter()
	setter(value)

	log.Printf(
		"0x%04X:\t%-12s ; %s = %s = 0x%02X", // More concise log
		cpu.opcodeAddr(instr),
		instr.String(),
		targetReg,
		sourceReg,
		value,
	)

	return instr.Cycles[0]
}

// OpPushR16 Handles PUSH rr (BC, DE, HL, AF)
func OpPushR16(cpu *CPU, instr *Instruction) int {
	if len(instr.Operands) < 1 {
		panic(fmt.Sprintf("OpPushR16 expects 1 operand, got 0 for %s", instr.String()))
	}
	sourceReg := instr.Operands[0].Name // "BC", "DE", "HL", "AF"
	var value uint16

	switch sourceReg {
	case "BC":
		value = cpu.Registers.getBC()
	case "DE":
		value = cpu.Registers.getDE()
	case "HL":
		value = cpu.Registers.getHL()
	case "AF":
		value = cpu.Registers.getAF()
	default:
		panic(fmt.Sprintf("OpPushR16: Unexpected source register %q", sourceReg))
	}

	cpu.pushWord(value)

	log.Printf(
		"0x%04X:\t%-12s ; pushed %s=0x%04X; SP=0x%04X",
		cpu.opcodeAddr(instr),
		instr.String(),
		sourceReg,
		value,
		cpu.Registers.getSP(),
	)
	return instr.Cycles[0]
}

func OpCbRlR8(cpu *CPU, instr *Instruction) int {
	if len(instr.Operands) < 1 {
		panic(fmt.Sprintf("OpCbRlR8 expects 1 operand, got 0 for %s", instr.String()))
	}
	targetReg := instr.Operands[0].Name
	getter := cpu.Registers.getRegisterGetter8(targetReg)
	setter := cpu.Registers.getRegisterSetter8(targetReg)
	if getter == nil || setter == nil {
		panic(fmt.Sprintf("OpCbRlR8: Unexpected register %q", targetReg))
	}

	old := getter()
	carryIn := byte(0)
	if cpu.Registers.getFlag(CarryFlag) {
		carryIn = 1
	}
	newCarry := (old>>7)&1 == 1
	result := (old<<1)&0xFE | carryIn
	setter(result)

	cpu.Registers.setFlag(ZeroFlag, result == 0)
	cpu.Registers.setFlag(SubtractFlag, false)
	cpu.Registers.setFlag(HalfCarryFlag, false)
	cpu.Registers.setFlag(CarryFlag, newCarry)

	log.Printf(
		"0x%04X:\t%-12s ; %s=0x%02X → 0x%02X ; Z=%t N=%t H=%t C=%t",
		cpu.opcodeAddr(instr),
		instr.String(),
		targetReg,
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

// OpPopR16 Handles POP rr (BC, DE, HL, AF)
func OpPopR16(cpu *CPU, instr *Instruction) int {
	if len(instr.Operands) < 1 {
		panic(fmt.Sprintf("OpPopR16 expects 1 operand, got 0 for %s", instr.String()))
	}
	targetReg := instr.Operands[0].Name // "BC", "DE", "HL", "AF"

	value := cpu.popWord()

	switch targetReg {
	case "BC":
		cpu.Registers.setBC(value)
	case "DE":
		cpu.Registers.setDE(value)
	case "HL":
		cpu.Registers.setHL(value)
	case "AF":
		cpu.Registers.setAF(value) // setAF handles masking lower F bits
	default:
		panic(fmt.Sprintf("OpPopR16: Unexpected target register %q", targetReg))
	}

	log.Printf(
		"0x%04X:\t%-12s ; popped 0x%04X → %s; SP=0x%04X",
		cpu.opcodeAddr(instr),
		instr.String(),
		value,
		targetReg,
		cpu.Registers.getSP(),
	)
	return instr.Cycles[0]
}
