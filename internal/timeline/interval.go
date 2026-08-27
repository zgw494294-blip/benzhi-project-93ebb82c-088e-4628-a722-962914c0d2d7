package timeline

import (
	"sort"

	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/domain"
)

type Interval struct {
	StartMS int64    `json:"start_ms"`
	EndMS   int64    `json:"end_ms"`
	Labels  []string `json:"labels,omitempty"`
}

func Normalize(items []Interval) ([]Interval, error) {
	copyItems := append([]Interval(nil), items...)
	for _, item := range copyItems {
		if err := domain.ValidateInterval(item.StartMS, item.EndMS, 0); err != nil {
			return nil, err
		}
	}
	sort.Slice(copyItems, func(i, j int) bool {
		if copyItems[i].StartMS == copyItems[j].StartMS {
			return copyItems[i].EndMS < copyItems[j].EndMS
		}
		return copyItems[i].StartMS < copyItems[j].StartMS
	})
	merged := make([]Interval, 0, len(copyItems))
	for _, item := range copyItems {
		if len(merged) == 0 || item.StartMS > merged[len(merged)-1].EndMS {
			item.Labels = uniqueStrings(item.Labels)
			merged = append(merged, item)
			continue
		}
		last := &merged[len(merged)-1]
		if item.EndMS > last.EndMS {
			last.EndMS = item.EndMS
		}
		last.Labels = uniqueStrings(append(last.Labels, item.Labels...))
	}
	return merged, nil
}

func uniqueStrings(items []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item != "" && !seen[item] {
			seen[item] = true
			out = append(out, item)
		}
	}
	sort.Strings(out)
	return out
}

func Overlaps(aStart, aEnd, bStart, bEnd int64) bool {
	return aStart < bEnd && bStart < aEnd
}
