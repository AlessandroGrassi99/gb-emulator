package main

import (
	"encoding/json"
	"fmt"
)

type JSONInstructions struct {
	Unprefixed map[string]Instruction `json:"unprefixed"`
	Cbprefixed map[string]Instruction `json:"cbprefixed"`
}

type Instructions struct {
	Unprefixed [256]Instruction
	Cbprefixed [256]Instruction
}

type Instruction struct {
	Name   string `json:"mnemonic"`
	Bytes  uint8  `json:"bytes"`
	Cycles []int  `json:"cycles"` // Cycles can have multiple values (e.g., for conditional jumps)
}

func (i *Instructions) UnmarshalJSON(data []byte) error {
	var raw JSONInstructions

	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("error unmarshaling into raw struct: %w", err)
	}

	for opcodeStr, ins := range raw.Unprefixed {
		hex, err := parseHexToUint8(opcodeStr)
		if err != nil {
			return fmt.Errorf("error processing unprefixed key %s: %w", opcodeStr, err)
		}
		i.Unprefixed[hex] = ins
	}
	for opcodeStr, ins := range raw.Cbprefixed {
		hex, err := parseHexToUint8(opcodeStr)
		if err != nil {
			return fmt.Errorf("error processing cbprefixed key %s: %w", opcodeStr, err)
		}
		i.Cbprefixed[hex] = ins
	}

	return nil
}

var instructionImplementations = map[string]func(*CPU) int{
	"NOP": nop,
}

func nop(cpu *CPU) int {
	panic(fmt.Sprintf("Generic LD handler called for opcode 0x%02X", 0))
}
