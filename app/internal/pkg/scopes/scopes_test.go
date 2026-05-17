package scopes

import (
	"errors"
	"testing"
	"time"
)

func TestParseTimeRange_Empty(t *testing.T) {
	from, to, err := ParseTimeRange("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if from != nil {
		t.Errorf("expected from nil, got %v", from)
	}
	if to != nil {
		t.Errorf("expected to nil, got %v", to)
	}
}

func TestParseTimeRange_StartOnly(t *testing.T) {
	from, to, err := ParseTimeRange("2024-01-15", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if to != nil {
		t.Errorf("expected to nil, got %v", to)
	}
	if from == nil {
		t.Fatal("expected from non-nil")
	}
	if from.Year() != 2024 || from.Month() != 1 || from.Day() != 15 {
		t.Errorf("expected 2024-01-15, got %v", from)
	}
}

func TestParseTimeRange_EndOnly_DateOnly(t *testing.T) {
	_, to, err := ParseTimeRange("", "2024-01-15")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if to == nil {
		t.Fatal("expected to non-nil")
	}
	h, m, s := to.Clock()
	if h != 23 || m != 59 || s != 59 {
		t.Errorf("expected end of day 23:59:59, got %02d:%02d:%02d", h, m, s)
	}
	if to.Nanosecond() != 999999999 {
		t.Errorf("expected nanosecond 999999999, got %d", to.Nanosecond())
	}
}

func TestParseTimeRange_EndOnly_DateTime(t *testing.T) {
	_, to, err := ParseTimeRange("", "2024-01-15T10:30:00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if to == nil {
		t.Fatal("expected to non-nil")
	}
	h, m, s := to.Clock()
	if h != 10 || m != 30 || s != 0 {
		t.Errorf("expected 10:30:00, got %02d:%02d:%02d", h, m, s)
	}
}

func TestParseTimeRange_BothDates(t *testing.T) {
	from, to, err := ParseTimeRange("2024-01-01", "2024-01-31")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if from == nil || to == nil {
		t.Fatal("expected both non-nil")
	}
	if !from.Before(*to) {
		t.Error("expected from < to")
	}
}

func TestParseTimeRange_RFC3339(t *testing.T) {
	from, to, err := ParseTimeRange("2024-01-15T08:00:00Z", "2024-01-15T18:00:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if from.Hour() != 8 || to.Hour() != 18 {
		t.Errorf("expected 8h and 18h, got %d and %d", from.Hour(), to.Hour())
	}
}

func TestParseTimeRange_StartAfterEnd(t *testing.T) {
	_, _, err := ParseTimeRange("2024-02-01", "2024-01-01")
	if err == nil {
		t.Fatal("expected error for start > end")
	}
	if !errors.Is(err, ErrTimeRangeOrder) {
		t.Errorf("expected ErrTimeRangeOrder, got %v", err)
	}
}

func TestParseTimeRange_InvalidStart(t *testing.T) {
	_, _, err := ParseTimeRange("not-a-date", "")
	if err == nil {
		t.Fatal("expected error for invalid start")
	}
	if !errors.Is(err, ErrStartTimeFormat) {
		t.Errorf("expected ErrStartTimeFormat, got %v", err)
	}
}

func TestParseTimeRange_InvalidEnd(t *testing.T) {
	_, _, err := ParseTimeRange("", "2024-13-01")
	if err == nil {
		t.Fatal("expected error for invalid end")
	}
	if !errors.Is(err, ErrEndTimeFormat) {
		t.Errorf("expected ErrEndTimeFormat, got %v", err)
	}
}

func TestParseTimeRange_SameDay(t *testing.T) {
	from, to, err := ParseTimeRange("2024-06-15", "2024-06-15")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	h, m, s := to.Clock()
	if h != 23 || m != 59 || s != 59 {
		t.Errorf("expected end of day 23:59:59, got %02d:%02d:%02d", h, m, s)
	}
	if to.Nanosecond() != 999999999 {
		t.Errorf("expected nanosecond 999999999, got %d", to.Nanosecond())
	}
	if !from.Before(*to) {
		t.Error("expected from < to for same day")
	}
}

func TestParseTimeRange_DateTimeSlash(t *testing.T) {
	from, _, err := ParseTimeRange("2024-01-15 10:30:00", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if from.Hour() != 10 || from.Minute() != 30 {
		t.Errorf("expected 10:30, got %02d:%02d", from.Hour(), from.Minute())
	}
}

func TestParseTimeRange_BoundaryMidnight(t *testing.T) {
	from, to, err := ParseTimeRange("2024-01-15", "2024-01-15")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if from.Hour() != 0 || from.Minute() != 0 || from.Second() != 0 {
		t.Errorf("expected start at midnight, got %v", from)
	}
	if to.Nanosecond() != 999999999 {
		t.Errorf("expected end at last nanosecond, got nanosecond=%d", to.Nanosecond())
	}
}

func TestTimeRange_Scope(t *testing.T) {
	now := time.Now()
	from := now.Add(-24 * time.Hour)
	to := now

	scope := TimeRange("created_at", &from, &to)
	if scope == nil {
		t.Fatal("expected non-nil scope")
	}
}

func TestTimeRange_ScopeNilBounds(t *testing.T) {
	scope := TimeRange("created_at", nil, nil)
	if scope == nil {
		t.Fatal("expected non-nil scope")
	}
}
