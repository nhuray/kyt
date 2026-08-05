package masker

import (
	"fmt"
	"strconv"
	"strings"
)

// Strategy represents a parsed masking strategy
type Strategy struct {
	Type      StrategyType
	KeepFirst int
	KeepLast  int
	MaskSeq   int // For sequenced strategy
	KeepMid   int // For sequenced strategy
}

// StrategyType defines the type of masking strategy
type StrategyType int

const (
	// StrategyKeepFirstLast keeps first F and last E characters, masks middle
	StrategyKeepFirstLast StrategyType = iota
	// StrategySequenced applies sequenced masking: keep F, keep E, then (mask S, keep M, repeat)
	StrategySequenced
)

// ParseStrategy parses a strategy string like "mask-2-2" or "mask-2-2-5-1"
func ParseStrategy(strategy string) (*Strategy, error) {
	parts := strings.Split(strategy, "-")
	if len(parts) < 3 || parts[0] != "mask" {
		return nil, fmt.Errorf("invalid strategy format: %s", strategy)
	}

	// Parse numbers
	var nums []int
	for i := 1; i < len(parts); i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return nil, fmt.Errorf("invalid number in strategy: %s", parts[i])
		}
		nums = append(nums, n)
	}

	if len(nums) == 2 {
		// mask-F-E
		return &Strategy{
			Type:      StrategyKeepFirstLast,
			KeepFirst: nums[0],
			KeepLast:  nums[1],
		}, nil
	} else if len(nums) == 4 {
		// mask-F-E-S-M
		return &Strategy{
			Type:      StrategySequenced,
			KeepFirst: nums[0],
			KeepLast:  nums[1],
			MaskSeq:   nums[2],
			KeepMid:   nums[3],
		}, nil
	}

	return nil, fmt.Errorf("invalid strategy: expected 2 or 4 numbers, got %d", len(nums))
}

// Execute applies the strategy to a value
func (s *Strategy) Execute(value, maskChar string) string {
	switch s.Type {
	case StrategyKeepFirstLast:
		return MaskKeepFirstLast(value, s.KeepFirst, s.KeepLast, maskChar)
	case StrategySequenced:
		return MaskSequenced(value, s.KeepFirst, s.KeepLast, s.MaskSeq, s.KeepMid, maskChar)
	default:
		return value // Should never happen
	}
}

// MaskKeepFirstLast masks middle, keeps first F and last E characters
func MaskKeepFirstLast(value string, keepFirst, keepLast int, maskChar string) string {
	if len(value) <= keepFirst+keepLast {
		// Value too short, mask everything
		return strings.Repeat(maskChar, len(value))
	}

	front := value[:keepFirst]
	back := value[len(value)-keepLast:]
	maskedLength := len(value) - keepFirst - keepLast
	masked := strings.Repeat(maskChar, maskedLength)

	return front + masked + back
}

// MaskSequenced applies sequenced masking: keep F, keep E, then (mask S, keep M, repeat)
// Example: "1234567890abcdefgh" with mask-2-2-5-1 → "12*****8*****d**gh"
func MaskSequenced(value string, keepFirst, keepLast, maskSeq, keepMid int, maskChar string) string {
	if len(value) <= keepFirst+keepLast {
		// Value too short, mask everything
		return strings.Repeat(maskChar, len(value))
	}

	// Split into: front | middle | back
	front := value[:keepFirst]
	back := value[len(value)-keepLast:]
	middle := value[keepFirst : len(value)-keepLast]

	// Apply sequence to middle
	var result strings.Builder
	result.WriteString(front)

	pos := 0
	for pos < len(middle) {
		// Mask S characters
		maskCount := maskSeq
		if pos+maskCount > len(middle) {
			maskCount = len(middle) - pos
			result.WriteString(strings.Repeat(maskChar, maskCount))
			break
		}
		result.WriteString(strings.Repeat(maskChar, maskCount))
		pos += maskCount

		if pos >= len(middle) {
			break
		}

		// Keep M characters (or all remaining if less than M)
		keepCount := keepMid
		if pos+keepCount > len(middle) {
			keepCount = len(middle) - pos
		}
		result.WriteString(middle[pos : pos+keepCount])
		pos += keepCount
	}

	result.WriteString(back)
	return result.String()
}
