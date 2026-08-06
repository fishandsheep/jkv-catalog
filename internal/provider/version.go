package provider

import (
	"regexp"
	"strconv"
)

var numericVersionPartRE = regexp.MustCompile(`[0-9]+`)

// compareNumericVersions compares version formats used by the reviewed providers.
// It treats every numeric component numerically, so 3.5.16 sorts after 3.5.9 and
// 17.0.15-b13 sorts after 17.0.15-b12.
func compareNumericVersions(left, right string) int {
	leftParts := numericVersionParts(left)
	rightParts := numericVersionParts(right)
	length := len(leftParts)
	if len(rightParts) > length {
		length = len(rightParts)
	}
	for index := 0; index < length; index++ {
		var leftPart, rightPart int
		if index < len(leftParts) {
			leftPart = leftParts[index]
		}
		if index < len(rightParts) {
			rightPart = rightParts[index]
		}
		if leftPart < rightPart {
			return -1
		}
		if leftPart > rightPart {
			return 1
		}
	}
	return 0
}

func numericVersionParts(version string) []int {
	raw := numericVersionPartRE.FindAllString(version, -1)
	parts := make([]int, 0, len(raw))
	for _, value := range raw {
		part, _ := strconv.Atoi(value)
		parts = append(parts, part)
	}
	return parts
}
