package main

import (
	"fmt"
	"strconv"
	"strings"
)

func parseHexToUint8(hex string) (uint8, error) {
	if !strings.HasPrefix(hex, "0x") {
		return 0, fmt.Errorf("invalid key format: expected '0x...' format, got %s", hex)
	}

	hexTrimmed := strings.TrimPrefix(hex, "0x")

	parsedKey, err := strconv.ParseUint(hexTrimmed, 16, 8)
	if err != nil {
		return 0, fmt.Errorf("error parsing key '%s' to uint8: %w", hexTrimmed, err)
	}

	return uint8(parsedKey), nil
}

func boolToUint8(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}
