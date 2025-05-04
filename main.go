//go:generate go run ./tools/gen_opcodes.go
//go:generate go run ./tools/gen_opcodes_dispatch.go
package main

import (
	_ "embed"
	"fmt"
)

//go:embed data/dmg_boot.bin
var bootROM []byte

func main() {
	for idx := range 512 {
		opFunc, ok := opcodesFunc[idx]
		if !ok {
			opcodes[idx].Execute = OpUnimplemented
		} else {
			opcodes[idx].Execute = opFunc
		}
	}

	cpu := &CPU{
		Mmu: NewMMU(),
		Registers: &Registers{
			PC: 0x0000,
			SP: 0xFFFE,
		},
	}

	fmt.Println("Starting emulation...")
	for {
		cpu.Step()
		//time.Sleep(100 * time.Millisecond)
	}
}
