package main

import "fmt"

type MMU struct {
	memory [0x10000]byte // 64KB address space
	// Add ROM loading, RAM, VRAM, I/O handling
}

func (mmu *MMU) ReadByte(addr uint16) byte {
	if int(addr) >= len(mmu.memory) {
		fmt.Printf("Warning: Read out of bounds at 0x%04X\n", addr)
		return 0xFF // returns FF for reads from non-existent memory
	}
	return mmu.memory[addr]
}

func (mmu *MMU) ReadWord(addr uint16) uint16 {
	low := uint16(mmu.ReadByte(addr))
	high := uint16(mmu.ReadByte(addr + 1))
	return (high << 8) | low
}
