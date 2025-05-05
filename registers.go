package main

const (
	ZeroFlag      uint8 = 1 << 7 // Z - bit 7
	SubtractFlag  uint8 = 1 << 6 // N - bit 6
	HalfCarryFlag uint8 = 1 << 5 // H - bit 5
	CarryFlag     uint8 = 1 << 4 // C bit 4
	// Lower 4 bits (3-0) of F are always 0
)

type Registers struct {
	A, F uint8 // Accumator and Flags
	B, C uint8 // General purpose
	D, E uint8 // General purpose
	H, L uint8 // General purpose

	PC uint16 // Program Counter
	SP uint16 // Stack Pointer
}

func (r *Registers) getA() uint8 {
	return r.A
}

func (r *Registers) setA(value uint8) {
	r.A = value
}

func (r *Registers) getAF() uint16 {
	return (uint16(r.A) << 8) | uint16(r.F)
}

func (r *Registers) setAF(value uint16) {
	r.A = uint8(value >> 8)
	r.F = uint8(value & 0x00F0)
}

func (r *Registers) getB() uint8 {
	return r.B
}

func (r *Registers) setB(value uint8) {
	r.B = value
}

func (r *Registers) getBC() uint16 {
	return (uint16(r.B) << 8) | uint16(r.C)
}

func (r *Registers) setBC(value uint16) {
	r.B = uint8(value >> 8)
	r.C = uint8(value)
}

func (r *Registers) getC() uint8 {
	return r.C
}

func (r *Registers) setC(value uint8) {
	r.C = value
}

func (r *Registers) getDE() uint16 {
	return (uint16(r.D) << 8) | uint16(r.E)
}

func (r *Registers) setDE(value uint16) {
	r.D = uint8(value >> 8)
	r.E = uint8(value)
}

func (r *Registers) getH() uint8 {
	return r.H
}

func (r *Registers) getHL() uint16 {
	return (uint16(r.H) << 8) | uint16(r.L)
}

func (r *Registers) setHL(value uint16) {
	r.H = uint8(value >> 8)
	r.L = uint8(value)
}

func (r *Registers) decHL() {
	value := r.getHL()
	r.setHL(value - 1)
}

func (r *Registers) getSP() uint16 {
	return r.SP
}

func (r *Registers) setSP(value uint16) {
	r.SP = value
}

func (r *Registers) getPC() uint16 {
	return r.PC
}

func (r *Registers) setPC(value uint16) {
	r.PC = value
}

func (r *Registers) addPC(value int8) {
	r.PC = uint16(int32(r.PC) + int32(value))
}

func (r *Registers) getFlag(flag uint8) bool {
	return (r.F & flag) != 0
}

func (r *Registers) setFlag(flag uint8, set bool) {
	if set {
		// set flag to 1 using the provided mask
		r.F |= flag
	} else {
		// inverse the mask, then AND to clear the specific bit
		r.F &= ^flag
	}
}

func (r *Registers) inc8Flags(orig, res byte) {
	r.setFlag(ZeroFlag, res == 0)
	r.setFlag(SubtractFlag, false)
	r.setFlag(HalfCarryFlag, (orig&0x0F) == 0x0F)
}
