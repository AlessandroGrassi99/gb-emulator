package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed opcodes.json
var opcodesRaw []byte
var opcodes Instructions

func main() {
	err := json.Unmarshal(opcodesRaw, &opcodes)
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println(opcodes.Unprefixed)

	// cpu := CPU{}

	// fmt.Println(cpu.Registers.getAF())
}
