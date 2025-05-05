package main

type CPU struct {
	Registers *Registers
	Mmu       *MMU
}

func (cpu *CPU) Step() int {
	instr := cpu.decode()

	cycles := instr.Execute(cpu, &instr)

	return cycles
}

func (cpu *CPU) decode() Instruction {
	opcode := cpu.fetchByte()

	var instr Instruction

	// CB prefixed
	if opcode == 0xCB {
		cbOpcode := cpu.fetchByte()
		instr = opcodes[256+int(cbOpcode)]
	} else {
		instr = opcodes[opcode]
	}

	return instr
}

func (cpu *CPU) fetchByte() uint8 {
	addr := cpu.Registers.PC
	val := cpu.Mmu.ReadByteAt(addr)
	cpu.Registers.PC++
	return val
}

func (cpu *CPU) fetchWord() uint16 {
	addr := cpu.Registers.PC
	val := cpu.Mmu.ReadWordAt(addr)
	cpu.Registers.PC += 2
	return val
}

func (cpu *CPU) opcodeAddr(instr *Instruction) uint16 {
	return cpu.Registers.getPC() - uint16(instr.Bytes)
}

func (c *CPU) pushWord(v uint16) {
	sp := c.Registers.getSP()
	sp--
	c.Mmu.WriteByteAt(sp, byte(v>>8)) // high
	sp--
	c.Mmu.WriteByteAt(sp, byte(v)) // low
	c.Registers.setSP(sp)
}

func (c *CPU) popWord() uint16 {
	sp := c.Registers.getSP()
	lo := uint16(c.Mmu.ReadByteAt(sp))
	sp++
	hi := uint16(c.Mmu.ReadByteAt(sp))
	sp++
	c.Registers.setSP(sp)
	return (hi << 8) | lo
}

func halfCarryAdd(a, b byte) bool { return ((a & 0xF) + (b & 0xF)) > 0xF }
