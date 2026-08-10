package job

import (
	"strings"
)

// NormalizeLocation cleans and normalizes job locations. If the raw location is generic
// like "Vietnam" or empty, it scans the job description for specific city indicators
// (e.g. Hanoi, Ho Chi Minh, Da Nang, district/street addresses).
func NormalizeLocation(rawLoc, description string) string {
	locLower := strings.ToLower(strings.TrimSpace(rawLoc))
	descLower := strings.ToLower(description)

	hcmKeywords := []string{
		"ho chi minh", "hcm", "tphcm", "tp.hcm", "tp. hcm",
		"sai gon", "saigon", "quan 1", "quan 3", "district 1", "district 3",
		"thu duc", "tan binh", "phu nhuan", "binh thanh",
	}

	hanoiKeywords := []string{
		"ha noi", "hanoi", "ha-noi", "cau giay", "ba dinh",
		"hoan kiem", "thanh xuan", "dong da", "hai ba trung", "nam tu liem", "bac tu liem",
	}

	danangKeywords := []string{
		"da nang", "danang", "da-nang", "hai chau", "ngu hanh son", "son tra", "thanh khe",
	}

	// 1. Check raw location first
	for _, kw := range hcmKeywords {
		if strings.Contains(locLower, kw) {
			return "Ho Chi Minh"
		}
	}
	for _, kw := range hanoiKeywords {
		if strings.Contains(locLower, kw) {
			return "Hanoi"
		}
	}
	for _, kw := range danangKeywords {
		if strings.Contains(locLower, kw) {
			return "Da Nang"
		}
	}

	if strings.Contains(locLower, "remote") || strings.Contains(locLower, "work from home") || strings.Contains(locLower, "wfh") {
		return "Remote"
	}

	// 2. If raw location is generic ("Vietnam", "viet nam", "vn", empty), fallback to description scan
	if locLower == "" || locLower == "vietnam" || locLower == "viet nam" || locLower == "vn" {
		for _, kw := range hcmKeywords {
			if strings.Contains(descLower, kw) {
				return "Ho Chi Minh"
			}
		}
		for _, kw := range hanoiKeywords {
			if strings.Contains(descLower, kw) {
				return "Hanoi"
			}
		}
		for _, kw := range danangKeywords {
			if strings.Contains(descLower, kw) {
				return "Da Nang"
			}
		}
		return "Vietnam"
	}

	return rawLoc
}
