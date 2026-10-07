package httpapi

import "time"

// An event comes back weekly, monthly or yearly; "" is a one-off. A birthday
// is always yearly. The page lays the occurrences out itself, and the backend
// needs only to tell whether a given day is one of them — an answer is for
// one day, and a day the series never lands on takes none.
var recurs = []string{"", "weekly", "monthly", "yearly"}

// occursOn says whether the series that starts on start (a date or a
// date-time, local wall clock) lands on day (YYYY-MM-DD). A monthly series
// started on the 31st lands on the last day of a shorter month, and a yearly
// one started on 29 February lands on the 28th in a year without one, so a
// month or a year never goes without its occurrence.
func occursOn(start, recur, day string) bool {
	s, err1 := time.Parse("2006-01-02", firstN(start, 10))
	d, err2 := time.Parse("2006-01-02", day)
	if err1 != nil || err2 != nil || d.Before(s) {
		return false
	}
	switch recur {
	case "":
		return d.Equal(s)
	case "weekly":
		return int(d.Sub(s).Hours()/24)%7 == 0
	case "monthly":
		return d.Day() == min(s.Day(), lastDay(d))
	case "yearly":
		return d.Month() == s.Month() && d.Day() == min(s.Day(), lastDay(d))
	}
	return false
}

func lastDay(d time.Time) int {
	return time.Date(d.Year(), d.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func firstN(s string, n int) string {
	if len(s) < n {
		return s
	}
	return s[:n]
}
