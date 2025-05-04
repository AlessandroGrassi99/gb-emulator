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
	opcode := cpu.fetchInstructionByte()

	var instr Instruction

	// CB prefixed
	if opcode == 0xCB {
		cbOpcode := cpu.fetchInstructionByte()
		instr = opcodes[256+int(cbOpcode)]
	} else {
		instr = opcodes[opcode]
	}

	return instr
}

func (cpu *CPU) fetchInstructionByte() uint8 {
	addr := cpu.Registers.PC
	val := cpu.Mmu.ReadByteAt(addr)
	cpu.Registers.PC++
	return val
}

func (cpu *CPU) fetchInstructionWord() uint16 {
	addr := cpu.Registers.PC
	val := cpu.Mmu.ReadWordAt(addr)
	cpu.Registers.PC += 2
	return val
}
