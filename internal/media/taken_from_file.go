package media

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/rwcarlsen/goexif/exif"
)

var (
	reISODateTime = regexp.MustCompile(`(20\d{2})[-_.]?(\d{2})[-_.]?(\d{2})(?:[ T_-](\d{2})[:._-]?(\d{2})[:._-]?(\d{2}))?`)
	reExifDT      = regexp.MustCompile(`^(\d{4}):(\d{2}):(\d{2})[ T](\d{2}):(\d{2}):(\d{2})`)
)

// TakenHint carries optional client-provided hints (tus metadata / filename).
type TakenHint struct {
	Filename string
	MTime    time.Time // zero if unknown
}

// ResolveTaken picks the best capture time for timeline grouping.
// Priority: EXIF / video meta → filename → client mtime → uploadNow.
func ResolveTaken(path, kind string, hint TakenHint, uploadNow time.Time, videoMetaISO string) (taken time.Time, date string, srcKind string) {
	uploadNow = uploadNow.UTC()
	if kind == "photo" {
		if t, ok := takenFromEXIF(path); ok {
			return noonUTC(t), t.Format("2006-01-02"), "exif"
		}
	}
	if kind == "video" {
		if t, ok := parseFlexibleTime(videoMetaISO); ok {
			return noonUTC(t), t.Format("2006-01-02"), "video_utc"
		}
	}
	name := hint.Filename
	if name == "" {
		name = filepath.Base(path)
	}
	if t, ok := takenFromFilename(name); ok {
		return noonUTC(t), t.Format("2006-01-02"), "upload"
	}
	if !hint.MTime.IsZero() {
		t := hint.MTime.UTC()
		return noonUTC(t), t.Format("2006-01-02"), "upload"
	}
	return noonUTC(uploadNow), uploadNow.Format("2006-01-02"), "upload"
}

func noonUTC(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 12, 0, 0, 0, time.UTC)
}

func takenFromEXIF(path string) (time.Time, bool) {
	f, err := os.Open(path)
	if err != nil {
		return time.Time{}, false
	}
	defer f.Close()
	x, err := exif.Decode(f)
	if err != nil {
		return time.Time{}, false
	}
	if t, err := x.DateTime(); err == nil {
		return t, true
	}
	for _, tag := range []exif.FieldName{exif.DateTimeOriginal, exif.DateTimeDigitized, exif.DateTime} {
		v, err := x.Get(tag)
		if err != nil {
			continue
		}
		s, err := v.StringVal()
		if err != nil {
			continue
		}
		if t, ok := parseExifDateTime(s); ok {
			return t, true
		}
	}
	return time.Time{}, false
}

func parseExifDateTime(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	m := reExifDT.FindStringSubmatch(s)
	if m == nil {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02 15:04:05", m[1]+"-"+m[2]+"-"+m[3]+" "+m[4]+":"+m[5]+":"+m[6])
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func takenFromFilename(name string) (time.Time, bool) {
	base := filepath.Base(name)
	m := reISODateTime.FindStringSubmatch(base)
	if m == nil {
		return time.Time{}, false
	}
	h, mi, s := "12", "00", "00"
	if m[4] != "" {
		h, mi, s = m[4], m[5], m[6]
	}
	t, err := time.Parse("2006-01-02 15:04:05", m[1]+"-"+m[2]+"-"+m[3]+" "+h+":"+mi+":"+s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func parseFlexibleTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000000Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}
