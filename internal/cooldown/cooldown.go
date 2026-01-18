package cooldown

import "time"

// Eligible reports whether a version published at updateTime is old enough
// given minDays. If minDays <= 0, it always returns true.
//
// If updateTime is empty or unparseable and minDays > 0, it returns false.
func Eligible(updateTime string, minDays int, now time.Time) bool {
	if minDays <= 0 {
		return true
	}
	if updateTime == "" {
		return false
	}

	t, err := time.Parse(time.RFC3339Nano, updateTime)
	if err != nil {
		var err2 error
		t, err2 = time.Parse(time.RFC3339, updateTime)
		if err2 != nil {
			return false
		}
	}

	age := now.Sub(t)
	if age < 0 {
		age = 0
	}
	minAge := time.Duration(minDays) * 24 * time.Hour
	return age >= minAge
}
