package main

type CPU struct {
	Registers *Registers
	Mmu       *MMU
}

// func (cpu *CPU) execute(opcode uint8) int {
// 	instr := opcodes.Unprefixed[opcode]
// }
