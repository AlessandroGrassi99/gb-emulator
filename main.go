//go:generate go run ./tools/gen_opcodes.go
package main

import (
	_ "embed"
	"fmt"
	"time"
)

func main() {
	for _, opcode := range opcodes {
		fmt.Println(opcode.String())
	}

	mmu := &MMU{}
	// Simple program:
	// 0x0100: LD BC, 0x1234
	// 0x0103: LD A, 0xAA
	// 0x0105: INC B
	// 0x0106: XOR A
	// 0x0107: JP 0x0100 (Infinite loop)
	mmu.memory[0x0100] = 0x01 // LD BC, d16
	mmu.memory[0x0101] = 0x34 // Low byte
	mmu.memory[0x0102] = 0x12 // High byte
	mmu.memory[0x0103] = 0x3E // LD A, d8
	mmu.memory[0x0104] = 0xAA // Value for A
	mmu.memory[0x0105] = 0x04 // INC B
	mmu.memory[0x0106] = 0xAF // XOR A
	mmu.memory[0x0107] = 0xC3 // JP a16
	mmu.memory[0x0108] = 0x00 // Low byte of jump target
	mmu.memory[0x0109] = 0x01 // High byte of jump target (-> 0x0100)

	registers := &Registers{
		PC: 0x0100,
		SP: 0xFFFE,
	}

	cpu := &CPU{
		Mmu:       mmu,
		Registers: registers,
	}

	fmt.Println("Starting emulation...")
	for range 20 {
		fmt.Printf("PC: 0x%04X | AF: 0x%04X BC: 0x%04X DE: 0x%04X HL: 0x%04X SP: 0x%04X\n",
			cpu.Registers.PC, cpu.Registers.getAF(), cpu.Registers.getBC(), cpu.Registers.getDE(), cpu.Registers.getHL(), cpu.Registers.SP)

		cpu.Step()

		time.Sleep(100 * time.Millisecond)
	}
}
