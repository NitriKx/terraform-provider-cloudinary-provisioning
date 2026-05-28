package util

import "time"

// DerefString returns the value pointed to by p, or "" if p is nil.
func DerefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// DerefBool returns the value pointed to by p, or false if p is nil.
func DerefBool(p *bool) bool {
	if p == nil {
		return false
	}
	return *p
}

// DerefTime returns the value pointed to by p, or the zero time if p is nil.
func DerefTime(p *time.Time) time.Time {
	if p == nil {
		return time.Time{}
	}
	return *p
}

// DerefInt64 returns the value pointed to by p, or 0 if p is nil.
func DerefInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}
