package stencil

import (
	"fmt"
	"strings"
	"time"
)

// Common date format patterns that we'll try to parse
var commonDateFormats = []string{
	// ISO and RFC formats
	time.RFC3339,
	time.RFC3339Nano,
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
	
	// Common formats
	"01/02/2006",
	"01/02/2006 15:04:05",
	"02/01/2006", // European style
	"2006/01/02",
	"2.1.2006",
	"02.01.2006",
	"2006.01.02",
	
	// Other formats
	"Jan 2, 2006",
	"January 2, 2006",
	"Mon, 02 Jan 2006",
	"Mon, 02 Jan 2006 15:04:05",
	"Monday, 02 January 2006",
}

// parseDate attempts to parse a date from various input types
func parseDate(value interface{}) (time.Time, error) {
	if value == nil {
		return time.Time{}, fmt.Errorf("cannot parse nil as date")
	}
	
	switch v := value.(type) {
	case time.Time:
		return v, nil
	case *time.Time:
		if v == nil {
			return time.Time{}, fmt.Errorf("cannot parse nil time pointer")
		}
		return *v, nil
	case int64:
		// Try as Unix timestamp (seconds)
		if v > 1e10 {
			// Likely milliseconds
			return time.Unix(v/1000, (v%1000)*1e6), nil
		}
		return time.Unix(v, 0), nil
	case int:
		return parseDate(int64(v))
	case float64:
		return parseDate(int64(v))
	case string:
		if v == "" {
			return time.Time{}, fmt.Errorf("cannot parse empty string as date")
		}
		
		// Try common formats
		for _, format := range commonDateFormats {
			if parsed, err := time.Parse(format, v); err == nil {
				return parsed, nil
			}
		}
		
		return time.Time{}, fmt.Errorf("could not parse date string: %s", v)
	default:
		// Try converting to string and parsing
		str := fmt.Sprintf("%v", v)
		return parseDate(str)
	}
}

// translateDateFormat converts Java SimpleDateFormat patterns to Go time format
func translateDateFormat(javaFormat string, _ string) string {
	// For now, we'll do a simple translation of common patterns
	// This is not exhaustive but covers most common cases
	// Note: locale parameter reserved for future locale-specific format translations
	
	format := javaFormat
	
	// Year
	format = strings.ReplaceAll(format, "yyyy", "2006")
	format = strings.ReplaceAll(format, "yy", "06")
	
	// Month
	format = strings.ReplaceAll(format, "MMMM", "January")
	format = strings.ReplaceAll(format, "MMM", "Jan")
	format = strings.ReplaceAll(format, "MM", "01")
	format = strings.ReplaceAll(format, "M", "1")
	
	// Day
	format = strings.ReplaceAll(format, "dd", "02")
	format = strings.ReplaceAll(format, "d", "2")
	
	// Hour (24-hour)
	format = strings.ReplaceAll(format, "HH", "15")
	format = strings.ReplaceAll(format, "H", "15")
	
	// Hour (12-hour)
	format = strings.ReplaceAll(format, "hh", "03")
	format = strings.ReplaceAll(format, "h", "3")
	
	// Minute
	format = strings.ReplaceAll(format, "mm", "04")
	format = strings.ReplaceAll(format, "m", "4")
	
	// Second
	format = strings.ReplaceAll(format, "ss", "05")
	format = strings.ReplaceAll(format, "s", "5")
	
	// AM/PM
	format = strings.ReplaceAll(format, "a", "PM")
	
	// Day of week
	format = strings.ReplaceAll(format, "EEEE", "Monday")
	format = strings.ReplaceAll(format, "EEE", "Mon")
	format = strings.ReplaceAll(format, "E", "Mon")
	
	// Timezone
	format = strings.ReplaceAll(format, "XXX", "Z07:00")
	format = strings.ReplaceAll(format, "XX", "Z0700")
	format = strings.ReplaceAll(format, "X", "Z07")
	format = strings.ReplaceAll(format, "zzz", "MST")
	format = strings.ReplaceAll(format, "Z", "Z0700")
	
	// Milliseconds
	format = strings.ReplaceAll(format, "SSS", "000")
	
	return format
}

// formatDateWithLocale formats a date with locale support
func formatDateWithLocale(t time.Time, format string, locale string) string {
	// First translate the format
	goFormat := translateDateFormat(format, locale)
	
	// Format the date
	result := t.Format(goFormat)
	
	// Apply locale-specific translations if needed
	if locale != "" && locale != "en" {
		result = applyLocaleTranslations(result, t, locale)
	}
	
	return result
}

// applyLocaleTranslations applies basic locale translations to formatted dates
func applyLocaleTranslations(formatted string, t time.Time, locale string) string {
	// Extract language from locale (e.g., "de-DE" -> "de")
	lang := locale
	if idx := strings.Index(locale, "-"); idx > 0 {
		lang = locale[:idx]
	}
	if idx := strings.Index(locale, "_"); idx > 0 {
		lang = locale[:idx]
	}
	
	// Get translations for the language
	translations := getDateTranslations(lang)
	if translations == nil {
		return formatted
	}
	
	// Apply month translations
	monthName := t.Format("January")
	if translated, ok := translations.months[monthName]; ok {
		formatted = strings.Replace(formatted, monthName, translated, -1)
	}
	
	monthShort := t.Format("Jan")
	if translated, ok := translations.monthsShort[monthShort]; ok {
		formatted = strings.Replace(formatted, monthShort, translated, -1)
	}
	
	// Apply weekday translations
	weekdayName := t.Format("Monday")
	if translated, ok := translations.weekdays[weekdayName]; ok {
		formatted = strings.Replace(formatted, weekdayName, translated, -1)
	}
	
	weekdayShort := t.Format("Mon")
	if translated, ok := translations.weekdaysShort[weekdayShort]; ok {
		formatted = strings.Replace(formatted, weekdayShort, translated, -1)
	}
	
	return formatted
}

type dateTranslations struct {
	months       map[string]string
	monthsShort  map[string]string
	weekdays     map[string]string
	weekdaysShort map[string]string
}

func getDateTranslations(lang string) *dateTranslations {
	switch lang {
	case "de": // German
		return &dateTranslations{
			months: map[string]string{
				"January": "Januar", "February": "Februar", "March": "März",
				"April": "April", "May": "Mai", "June": "Juni",
				"July": "Juli", "August": "August", "September": "September",
				"October": "Oktober", "November": "November", "December": "Dezember",
			},
			monthsShort: map[string]string{
				"Jan": "Jan", "Feb": "Feb", "Mar": "Mär",
				"Apr": "Apr", "May": "Mai", "Jun": "Jun",
				"Jul": "Jul", "Aug": "Aug", "Sep": "Sep",
				"Oct": "Okt", "Nov": "Nov", "Dec": "Dez",
			},
			weekdays: map[string]string{
				"Monday": "Montag", "Tuesday": "Dienstag", "Wednesday": "Mittwoch",
				"Thursday": "Donnerstag", "Friday": "Freitag",
				"Saturday": "Samstag", "Sunday": "Sonntag",
			},
			weekdaysShort: map[string]string{
				"Mon": "Mo", "Tue": "Di", "Wed": "Mi",
				"Thu": "Do", "Fri": "Fr", "Sat": "Sa", "Sun": "So",
			},
		}
		
	case "fr": // French
		return &dateTranslations{
			months: map[string]string{
				"January": "janvier", "February": "février", "March": "mars",
				"April": "avril", "May": "mai", "June": "juin",
				"July": "juillet", "August": "août", "September": "septembre",
				"October": "octobre", "November": "novembre", "December": "décembre",
			},
			monthsShort: map[string]string{
				"Jan": "jan", "Feb": "fév", "Mar": "mar",
				"Apr": "avr", "May": "mai", "Jun": "juin",
				"Jul": "juil", "Aug": "août", "Sep": "sep",
				"Oct": "oct", "Nov": "nov", "Dec": "déc",
			},
			weekdays: map[string]string{
				"Monday": "lundi", "Tuesday": "mardi", "Wednesday": "mercredi",
				"Thursday": "jeudi", "Friday": "vendredi",
				"Saturday": "samedi", "Sunday": "dimanche",
			},
			weekdaysShort: map[string]string{
				"Mon": "lun", "Tue": "mar", "Wed": "mer",
				"Thu": "jeu", "Fri": "ven", "Sat": "sam", "Sun": "dim",
			},
		}
		
	case "es": // Spanish
		return &dateTranslations{
			months: map[string]string{
				"January": "enero", "February": "febrero", "March": "marzo",
				"April": "abril", "May": "mayo", "June": "junio",
				"July": "julio", "August": "agosto", "September": "septiembre",
				"October": "octubre", "November": "noviembre", "December": "diciembre",
			},
			monthsShort: map[string]string{
				"Jan": "ene", "Feb": "feb", "Mar": "mar",
				"Apr": "abr", "May": "may", "Jun": "jun",
				"Jul": "jul", "Aug": "ago", "Sep": "sep",
				"Oct": "oct", "Nov": "nov", "Dec": "dic",
			},
			weekdays: map[string]string{
				"Monday": "lunes", "Tuesday": "martes", "Wednesday": "miércoles",
				"Thursday": "jueves", "Friday": "viernes",
				"Saturday": "sábado", "Sunday": "domingo",
			},
			weekdaysShort: map[string]string{
				"Mon": "lun", "Tue": "mar", "Wed": "mié",
				"Thu": "jue", "Fri": "vie", "Sat": "sáb", "Sun": "dom",
			},
		}
		
	case "it": // Italian
		return &dateTranslations{
			months: map[string]string{
				"January": "gennaio", "February": "febbraio", "March": "marzo",
				"April": "aprile", "May": "maggio", "June": "giugno",
				"July": "luglio", "August": "agosto", "September": "settembre",
				"October": "ottobre", "November": "novembre", "December": "dicembre",
			},
			monthsShort: map[string]string{
				"Jan": "gen", "Feb": "feb", "Mar": "mar",
				"Apr": "apr", "May": "mag", "Jun": "giu",
				"Jul": "lug", "Aug": "ago", "Sep": "set",
				"Oct": "ott", "Nov": "nov", "Dec": "dic",
			},
			weekdays: map[string]string{
				"Monday": "lunedì", "Tuesday": "martedì", "Wednesday": "mercoledì",
				"Thursday": "giovedì", "Friday": "venerdì",
				"Saturday": "sabato", "Sunday": "domenica",
			},
			weekdaysShort: map[string]string{
				"Mon": "lun", "Tue": "mar", "Wed": "mer",
				"Thu": "gio", "Fri": "ven", "Sat": "sab", "Sun": "dom",
			},
		}
		
	default:
		return nil
	}
}

// registerDateFunctions registers all date-related functions
func registerDateFunctions(registry *DefaultFunctionRegistry) {
	// date() function - formats a date object
	dateFn := NewSimpleFunction("date", 2, 3, func(args ...interface{}) (interface{}, error) {
		// Handle nil arguments
		if len(args) == 2 {
			if args[0] == nil || args[1] == nil {
				return nil, nil
			}
			if str, ok := args[1].(string); ok && str == "" {
				return nil, nil
			}
		} else if len(args) == 3 {
			if args[0] == nil || args[1] == nil || args[2] == nil {
				return nil, nil
			}
			if str, ok := args[2].(string); ok && str == "" {
				return nil, nil
			}
		}
		
		var locale, pattern string
		var dateValue interface{}
		
		if len(args) == 2 {
			// date(pattern, value)
			locale = ""
			pattern = FormatValue(args[0])
			dateValue = args[1]
		} else {
			// date(locale, pattern, value)
			locale = FormatValue(args[0])
			pattern = FormatValue(args[1])
			dateValue = args[2]
		}
		
		// Parse the date
		t, err := parseDate(dateValue)
		if err != nil {
			return nil, fmt.Errorf("could not parse date object: %v", err)
		}
		
		// Check if the pattern is a valid Go format by trying it
		// If it fails, it might be a Java-style format that needs translation
		testResult := t.Format(pattern)
		if strings.Contains(testResult, pattern) || len(testResult) == 0 {
			// Format failed, likely a Java-style pattern
			return formatDateWithLocale(t, pattern, locale), nil
		}
		
		// Go format worked directly
		if locale != "" && locale != "en" {
			return applyLocaleTranslations(testResult, t, locale), nil
		}
		return testResult, nil
	})
	registry.RegisterFunction(dateFn)
}