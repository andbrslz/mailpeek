package mail

import (
	"io"
	"strings"
	"unicode/utf8"
)

var cp1252 = [32]rune{
	0x20AC, 0, 0x201A, 0x0192, 0x201E, 0x2026, 0x2020, 0x2021,
	0x02C6, 0x2030, 0x0160, 0x2039, 0x0152, 0, 0x017D, 0,
	0, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2013, 0x2014,
	0x02DC, 0x2122, 0x0161, 0x203A, 0x0153, 0, 0x017E, 0x0178,
}

func toUTF8(charset string, b []byte) string {
	switch strings.ToLower(strings.TrimSpace(charset)) {
	case "iso-8859-1", "iso8859-1", "latin1", "l1":
		return decodeSingleByte(b, false)
	case "windows-1252", "cp1252":
		return decodeSingleByte(b, true)
	}
	if utf8.Valid(b) {
		return string(b)
	}
	return strings.ToValidUTF8(string(b), "�")
}

func decodeSingleByte(b []byte, windows bool) string {
	var sb strings.Builder
	sb.Grow(len(b))
	for _, c := range b {
		r := rune(c)
		if windows && c >= 0x80 && c <= 0x9F {
			if r = cp1252[c-0x80]; r == 0 {
				r = utf8.RuneError
			}
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

func charsetReader(charset string, input io.Reader) (io.Reader, error) {
	b, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}
	return strings.NewReader(toUTF8(charset, b)), nil
}
