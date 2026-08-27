package timeline

import (
	"math"
	"unicode"
)

func CountReadableCharacters(text string) int {
	count := 0
	for _, r := range text {
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			continue
		}
		count++
	}
	return count
}

func EstimateReadingMS(text string, charsPerSecond float64, pauseBudgetMS int64) int64 {
	if charsPerSecond <= 0 {
		return math.MaxInt64
	}
	chars := CountReadableCharacters(text)
	spoken := int64(math.Ceil(float64(chars) / charsPerSecond * 1000))
	if pauseBudgetMS < 0 {
		pauseBudgetMS = 0
	}
	return spoken + pauseBudgetMS
}

func FormatTimecode(ms int64) string {
	if ms < 0 {
		ms = 0
	}
	hours := ms / 3600000
	ms %= 3600000
	minutes := ms / 60000
	ms %= 60000
	seconds := ms / 1000
	millis := ms % 1000
	return two(hours) + ":" + two(minutes) + ":" + two(seconds) + "." + three(millis)
}

func two(v int64) string {
	return string([]byte{'0' + byte((v/10)%10), '0' + byte(v%10)})
}

func three(v int64) string {
	return string([]byte{'0' + byte((v/100)%10), '0' + byte((v/10)%10), '0' + byte(v%10)})
}
