package main

import "fmt"

const (
	bootDisableReg = 0xFF50
)

type MMU struct {
	memory      [0x10000]byte // 64KB address space
	boot        [0x00100]byte // 256B address space
	bootEnabled bool
}

func NewMMU() *MMU {
	m := &MMU{bootEnabled: true}
	copy(m.boot[:], bootROM)
	return m
}

func (mmu *MMU) ReadByteAt(addr uint16) uint8 {
	// While enabled, 0x0000–0x00FF is served by the internal ROM
	if mmu.bootEnabled && addr < 0x0100 {
		return mmu.boot[addr]
	}

	if int(addr) >= len(mmu.memory) {
		fmt.Printf("Warning: Read memory out of bounds at 0x%04X\n", addr)
		return 0xFF
	}
	return mmu.memory[addr]
}

func (mmu *MMU) ReadWordAt(addr uint16) uint16 {
	lo := uint16(mmu.ReadByteAt(addr))
	hi := uint16(mmu.ReadByteAt(addr + 1))
	return (hi << 8) | lo
}

func (mmu *MMU) WriteByteAt(addr uint16, value uint8) {
	// Permanently switch boot ROM out of 0x0000‑0x00FF
	if addr == bootDisableReg {
		// Latch boot ROM off permanently if bit0 == 1
		if value&0x01 != 0 {
			mmu.bootEnabled = false
		}
		// Store the value so games/tests can read it back
		mmu.memory[addr] = value
		return
	}
	if int(addr) >= len(mmu.memory) {
		fmt.Printf("Warning: Write memory out of bounds at 0x%04X\n", addr)
		return
	}
	mmu.memory[addr] = value
}
