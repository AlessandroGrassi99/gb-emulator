package main

import "fmt"

type MMU struct {
	memory      [0x10000]byte // 64KB address space
	bootEnabled bool
	boot        [0x00100]byte // 256B address space
}

func NewMMU() *MMU {
	m := &MMU{bootEnabled: true}
	copy(m.boot[:], bootROM)
	return m
}

func (mmu *MMU) ReadByteAt(addr uint16) uint8 {
	// While the boot ROM is enabled, only 0x0000–0x00FF should come from boot[]
	if mmu.bootEnabled && int(addr) < len(mmu.boot) {
		return mmu.boot[addr]
	}

	// Otherwise fall back to normal memory
	if int(addr) >= len(mmu.memory) {
		fmt.Printf("Warning: Read memory out of bounds at 0x%04X\n", addr)
		return 0xFF
	}
	return mmu.memory[addr]
}

func (mmu *MMU) ReadWordAt(addr uint16) uint16 {
	low := uint16(mmu.ReadByteAt(addr))
	high := uint16(mmu.ReadByteAt(addr + 1))
	return (high << 8) | low
}

func (mmu *MMU) WriteByteAt(addr uint16, value uint8) {
	if int(addr) >= len(mmu.memory) {
		fmt.Printf("Warning: Write memory out of bounds at 0x%04X\n", addr)
		return
	}
	mmu.memory[addr] = value
}
