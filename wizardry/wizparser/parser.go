package wizparser

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"github.com/yizhibenpao/go-filetype/wizardry/wizutil"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"
)

// LogFunc prints a debug message
type LogFunc func(format string, args ...interface{})

// ParseContext holds state for the parser
type ParseContext struct {
	Logf LogFunc
}

func convertUTF16ToLittleEndianBytes(u []uint16) []byte {
	b := make([]byte, 2*len(u))
	for index, value := range u {
		binary.LittleEndian.PutUint16(b[index*2:], value)
	}
	return b
}
func convertUTF16ToBigEndianBytes(u []uint16) []byte {
	b := make([]byte, 2*len(u))
	for index, value := range u {
		binary.BigEndian.PutUint16(b[index*2:], value)
	}
	return b
}

// ParseAll parses all the files in a directory and adds them to the same spellbook
func (ctx *ParseContext) ParseAll(magdir string, book Spellbook) error {
	files, err := ioutil.ReadDir(magdir)
	if err != nil {
		return errors.WithStack(err)
	}

	for _, magicFile := range files {
		err = func() error {
			f, err := os.Open(filepath.Join(magdir, magicFile.Name()))
			if err != nil {
				return errors.WithStack(err)
			}

			defer f.Close()

			err = ctx.Parse(f, book)
			if err != nil {
				return errors.WithStack(err)
			}

			return nil
		}()

		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// Parse reads a magic rule file and puts it into a spell book
func (ctx *ParseContext) Parse(magicReader io.Reader, book Spellbook) error {
	scanner := bufio.NewScanner(magicReader)

	page := ""

	for scanner.Scan() {
		line := scanner.Text()
		lineBytes := []byte(line)
		numBytes := len(lineBytes)

		if numBytes == 0 {
			// empty line, ignore
			continue
		}

		i := 0

		if lineBytes[i] == '#' {
			// comment, ignore
			continue
		}

		if lineBytes[i] == '!' {
			continue
		}

		rule := Rule{}

		rule.Line = line

		// read level
		for i < numBytes && lineBytes[i] == '>' {
			rule.Level++
			i++
		}

		if rule.Level < 1 {
			// end of the page, if any
			if page != "" {
				ctx.Logf("end of page %s", page)
				page = ""
			}
		}

		ctx.Logf("| %s", line)

		// read offset
		offsetStart := i
		for i < numBytes && !wizutil.IsWhitespace(lineBytes[i]) {
			i++
		}
		offsetEnd := i
		offset := line[offsetStart:offsetEnd]

		// skip whitespace
		for i < numBytes && wizutil.IsWhitespace(lineBytes[i]) {
			i++
		}

		// read kind
		kindStart := i
		for i < numBytes && !wizutil.IsWhitespace(lineBytes[i]) {
			i++
		}
		kindEnd := i
		kind := lineBytes[kindStart:kindEnd]

		// skip whitespace
		for i < numBytes && wizutil.IsWhitespace(lineBytes[i]) {
			i++
		}

		// read test
		testStart := i
		for i < numBytes && !wizutil.IsWhitespace(lineBytes[i]) {
			// this isn't the greatest trick in the world tbh
			if lineBytes[i] == '\\' {
				i += 2
			} else {
				i++
			}
		}
		testEnd := i
		test := lineBytes[testStart:testEnd]
		//test_str in ("<", ">", "=", "!", "&", "^", "~")
		if string(test) == "<" || string(test) == ">" || string(test) == "=" || string(test) == "!" || string(test) == "&" || string(test) == "^" || string(test) == "~" {
			for i < numBytes && wizutil.IsWhitespace(lineBytes[i]) {
				i++
			}
			testsecStart := i
			for i < numBytes && !wizutil.IsWhitespace(lineBytes[i]) {
				// this isn't the greatest trick in the world tbh
				if lineBytes[i] == '\\' {
					i += 2
				} else {
					i++
				}
			}
			testsecEnd := i
			test = append(test, lineBytes[testsecStart:testsecEnd]...)

		}

		// skip whitespace
		for i < numBytes && wizutil.IsWhitespace(lineBytes[i]) {
			i++
		}

		descriptionBytes := lineBytes[i:]

		// parse offset
		{
			offsetBytes := []byte(offset)
			j := 0
			if offsetBytes[j] == '&' {
				// offset is relative to globalOffset
				rule.Offset.IsRelative = true
				j++
			}

			if offsetBytes[j] == '(' {
				j++
				rule.Offset.OffsetType = OffsetTypeIndirect

				indirect := &IndirectOffset{}
				rule.Offset.Indirect = indirect

				if offsetBytes[j] == '&' {
					indirect.IsRelative = true
					j++
				}

				indirectAddr, err := parseInt(offsetBytes, j)
				if err != nil {
					ctx.Logf("error: couldn't parse indirect offset in part \"%s\" of rule %s", offsetBytes[j:], line)
					continue
				}

				j = indirectAddr.NewIndex

				indirect.OffsetAddress = indirectAddr.Value
				indirectAddrFormat := byte('I')
				if offsetBytes[j] != '.' && offsetBytes[j] != ',' {
					indirectAddrFormat = byte('I')
					//ctx.Logf("malformed indirect offset in %s, expected [.,], got '%c'\n", string(offsetBytes), offsetBytes[j])
					//continue
				} else {
					j++

					indirectAddrFormat = offsetBytes[j]
					j++
				}

				indirect.Endianness = LittleEndian

				if wizutil.IsUpperLetter(indirectAddrFormat) {
					indirect.Endianness = BigEndian
					indirectAddrFormat = wizutil.ToLower(indirectAddrFormat)
				}

				switch indirectAddrFormat {
				case 'b', 'c':
					indirect.ByteWidth = 1
				case 'i':
					indirect.ByteWidth = 4
					//ctx.Logf("id3 format not supported, skipping %s", line)
					//continue
				case 's', 'h':
					indirect.ByteWidth = 2
				case 'l':
					indirect.ByteWidth = 4
				case 'q', 'e', 'f', 'g':
					indirect.ByteWidth = 8
				case 'm':
					ctx.Logf("middle-endian format not supported, skipping %s", line)
					continue
				default:
					ctx.Logf("unsupported indirect addr format %c, skipping %s", indirectAddrFormat, line)
					continue
				}

				if offsetBytes[j] == '+' {
					indirect.OffsetAdjustmentType = AdjustmentAdd
				} else if offsetBytes[j] == '-' {
					indirect.OffsetAdjustmentType = AdjustmentSub
				} else if offsetBytes[j] == '*' {
					indirect.OffsetAdjustmentType = AdjustmentMul
				} else if offsetBytes[j] == '/' {
					indirect.OffsetAdjustmentType = AdjustmentDiv
				} else if offsetBytes[j] == '&' {
					indirect.OffsetAdjustmentType = AdjustmentAnd
				}

				if indirect.OffsetAdjustmentType != AdjustmentNone {
					j++
					// it's a relative pair
					if offsetBytes[j] == '(' {
						indirect.OffsetAdjustmentIsRelative = true
						j++
					}

					parsedRHS, err := parseInt(offsetBytes, j)
					if err != nil {
						ctx.Logf("malformed indirect offset rhs, skipping %s", line)
						continue
					}

					indirect.OffsetAdjustmentValue = parsedRHS.Value
					j = parsedRHS.NewIndex

					if indirect.OffsetAdjustmentIsRelative {
						if offsetBytes[j] != ')' {
							ctx.Logf("malformed relative offset adjustment, missing closing ')' - in %s", line)
							continue
						}
						j++
					}
				}

				if offsetBytes[j] != ')' {
					ctx.Logf("malformed indirect offset in %s, expected ')', got '%c', skipping", string(offsetBytes), offsetBytes[j])
					continue
				}
				j++
			} else {
				rule.Offset.OffsetType = OffsetTypeDirect

				parsedAbsolute, err := parseInt(offsetBytes, j)
				if err != nil {
					ctx.Logf("malformed absolute offset, expected number, got (%s), skipping", offsetBytes[j:])
					continue
				}

				rule.Offset.Direct = parsedAbsolute.Value
				j = parsedAbsolute.NewIndex
			}
		}

		// parse kind
		{
			j := 0
			parsedKind := parseKind(kind, j)
			j += parsedKind.NewIndex

			switch parsedKind.Value {
			case
				"ubyte", "ushort", "ulong", "uquad",
				"ubeshort", "ubelong", "ubequad",
				"uleshort", "ulelong", "ulequad",
				"byte", "short", "long", "quad",
				"beshort", "belong", "bequad",
				"leshort", "lelong", "lequad", "bedate", "beldate", "beqdate", "beqldate",
				"ldate", "ledate", "leldate", "lemsdosdate", "lemsdostime",
				"leqdate", "leqldate", "leqwdate", "medate", "ubeqdate",
				"bedouble", "ledouble", "lefloat":

				ik := &IntegerKind{}
				rule.Kind.Family = KindFamilyInteger
				rule.Kind.Data = ik

				ik.Signed = true
				ik.Endianness = LittleEndian

				simpleKind := parsedKind.Value
				if strings.HasPrefix(simpleKind, "u") {
					simpleKind = simpleKind[1:]
					ik.Signed = false
				}

				if strings.HasPrefix(simpleKind, "le") {
					simpleKind = simpleKind[2:]
				} else if strings.HasPrefix(simpleKind, "be") {
					simpleKind = simpleKind[2:]
					ik.Endianness = BigEndian
				}

				switch simpleKind {
				case "byte":
					ik.ByteWidth = 1
				case "short":
					ik.ByteWidth = 2
				case "long", "date", "ldate", "msdosdate", "msdostime", "float":
					ik.ByteWidth = 4
				case "quad", "qdate", "qldate", "qwdate", "double":
					ik.ByteWidth = 8
				default:
					ctx.Logf("unrecognized integer kind %s, skipping rule %s", simpleKind, line)
					continue
				}

				ik.DoAnd = false

				if j < len(kind) {
					switch kind[j] {
					case '+':
						ik.AdjustmentType = AdjustmentAdd
						j++
					case '-':
						ik.AdjustmentType = AdjustmentSub
						j++
					case '*':
						ik.AdjustmentType = AdjustmentMul
						j++
					case '/':
						ik.AdjustmentType = AdjustmentDiv
						j++
					case '&':
						ik.AdjustmentType = AdjustmentAnd
						j++
					}

					if ik.AdjustmentType != AdjustmentNone && ik.AdjustmentType != AdjustmentAnd {
						pi, err := parseInt(kind, j)
						if err != nil {
							ctx.Logf("couldn't parser integer kind adjustment in %s, skipping rule %s", kind[j:], line)
							continue
						}
						ik.AdjustmentValue = pi.Value
						j = pi.NewIndex
					}
				}

				if j < len(kind) && ik.AdjustmentType == '&' {
					parsedAndValue, err := parseUint(kind, j)
					if err != nil {
						ctx.Logf("in integer test, couldn't parse and value %s, skipping\n", kind[j:])
						continue
					}
					ik.DoAnd = true
					ik.AndValue = parsedAndValue.Value
					j = parsedAndValue.NewIndex
				}

				ik.IntegerTest = IntegerTestEqual

				k := 0

				switch test[k] {
				case 'x':
					ik.MatchAny = true
					k++
				case '=':
					ik.IntegerTest = IntegerTestEqual
					k++
				case '!':
					ik.IntegerTest = IntegerTestNotEqual
					k++
				case '<':
					ik.IntegerTest = IntegerTestLessThan
					k++
				case '>':
					ik.IntegerTest = IntegerTestGreaterThan
					k++
				case '&':
					ik.IntegerTest = IntegerTestAnd
					k++
				case '^':
					ik.IntegerTest = IntegerTestNOTAnd
					k++
				}

				if !ik.MatchAny {
					parsedMagicintValue, errint := parseInt(test, k)
					parsedMagicValue, err := parseUint(test, k)
					if err == nil {
						ik.Value = parsedMagicValue.Value
						k = parsedMagicValue.NewIndex

					}
					if errint == nil {
						ik.Value = uint64(parsedMagicintValue.Value)
						k = parsedMagicintValue.NewIndex

					}
					if errint != nil && err != nil {
						ctx.Logf("for integer test, couldn't parse magic value %s, ignoring", string(test[k:]))
						continue
					}
					//parsedMagicValue, err := parseInt(test, k)
					//if err != nil {
					//	ctx.Logf("for integer test, couldn't parse magic value %s, ignoring", string(test[k:]))
					//	continue
					//}
					//
					//ik.Value = parsedMagicValue.Value
					//k = parsedMagicValue.NewIndex

				}
			case "indirect":
			case "string":
				sk := &StringKind{}
				rule.Kind.Family = KindFamilyString
				rule.Kind.Data = sk

				k := 0
				sk.Negate = false
				if test[k] == '!' {
					sk.Negate = true
					k++
				}

				parsedRHS, err := parseString(test, k)
				if err != nil {
					ctx.Logf("in string test, couldn't parse rhs: %s - skipping", err.Error())
					continue
				}
				sk.Value = parsedRHS.Value

				if j < len(kind) && kind[j] == '/' {
					j++
					parsedFlags := parseStringTestFlags(kind, j)
					j = parsedFlags.NewIndex
					sk.Flags = parsedFlags.Flags
				}
			case "guid":
				sk := &GUIDKind{}
				rule.Kind.Family = KindFamilyGUID
				rule.Kind.Data = sk
				sk.Value = test
			case "lestring16":
				sk := &StringKind{}
				rule.Kind.Family = KindFamilyString
				rule.Kind.Data = sk

				k := 0
				sk.Negate = false
				if test[k] == '!' {
					sk.Negate = true
					k++
				}

				parsedRHS, err := parseString(test, k)
				if err != nil {
					ctx.Logf("in string test, couldn't parse rhs: %s - skipping", err.Error())
					continue
				}
				if string(parsedRHS.Value) == "x" {
					sk.Value = parsedRHS.Value
				} else {
					encodeed := utf16.Encode([]rune(string(parsedRHS.Value)))
					sk.Value = convertUTF16ToLittleEndianBytes(encodeed)
				}

				if j < len(kind) && kind[j] == '/' {
					j++
					parsedFlags := parseStringTestFlags(kind, j)
					j = parsedFlags.NewIndex
					sk.Flags = parsedFlags.Flags
				}
			case "bestring16":
				sk := &StringKind{}
				rule.Kind.Family = KindFamilyString
				rule.Kind.Data = sk

				k := 0
				sk.Negate = false
				if test[k] == '!' {
					sk.Negate = true
					k++
				}

				parsedRHS, err := parseString(test, k)
				if err != nil {
					ctx.Logf("in string test, couldn't parse rhs: %s - skipping", err.Error())
					continue
				}
				if string(parsedRHS.Value) == "x" {
					sk.Value = parsedRHS.Value
				} else {
					encodeed := utf16.Encode([]rune(string(parsedRHS.Value)))
					sk.Value = convertUTF16ToBigEndianBytes(encodeed)
				}

				if j < len(kind) && kind[j] == '/' {
					j++
					parsedFlags := parseStringTestFlags(kind, j)
					j = parsedFlags.NewIndex
					sk.Flags = parsedFlags.Flags
				}

			case "regex":
				rk := &RegexKind{}
				rule.Kind.Family = KindFamilyRegex
				rule.Kind.Data = rk

				k := 0
				parsedRHR, err := parseRegex(test, k)
				if err != nil {
					ctx.Logf("in string test, couldn't parse rhs: %s - skipping", err.Error())
					continue
				}
				rk.Value = parsedRHR.Value
				rk.MaxLen = 8192
				if j < len(kind) && kind[j] == '/' {
					j++
					parsedFlags := parseRegexTestFlags(kind, j)
					j = parsedFlags.NewIndex
					rk.Flags = parsedFlags.Flags
					if parsedFlags.Maxlen > 0 {
						rk.MaxLen = parsedFlags.Maxlen
					}

				}

			case "search":
				sk := &SearchKind{}
				rule.Kind.Family = KindFamilySearch
				rule.Kind.Data = sk

				sk.MaxLen = 8192
				if j < len(kind) && kind[j] == '/' {
					j++
					if kind[j] == 'b' {
						j++
						if kind[j] == '/' {
							j++
						}
					}
					parsedLen, err := parseUint(kind, j)
					if err != nil {
						ctx.Logf("in search test, couldn't parse max len in %s: %s - skipping\n", kind[j:], err.Error())
						continue
					}

					j = parsedLen.NewIndex
					sk.MaxLen = int64(parsedLen.Value)
				}

				k := 0

				parsedRHS, err := parseString(test, k)
				if err != nil {
					fmt.Printf("in search test, couldn't parse rhs: %s - skipping", err.Error())
					continue
				}
				k = parsedRHS.NewIndex
				sk.Value = parsedRHS.Value
			case "pstring":
				sk := &PStringKind{}
				rule.Kind.Family = KindFamilyPString
				rule.Kind.Data = sk

				sk.LengthByte = 1
				sk.ContainsOwnLength = false
				sk.EndiannessBIG = false

				if j < len(kind) && kind[j] == '/' {
					j++
					for {
						switch kind[j] {
						case 'B':
							sk.LengthByte = 1
						case 'H':
							sk.LengthByte = 2
							sk.EndiannessBIG = true
						case 'h':
							sk.LengthByte = 2
						case 'L':
							sk.LengthByte = 4
							sk.EndiannessBIG = true
						case 'l':
							sk.LengthByte = 4
						case 'J':
							sk.ContainsOwnLength = true
						}
						j++
						if j >= len(kind) {
							break
						}
					}
					j = 0
				}

				k := 0

				parsedRHS, err := parseString(test, k)
				if err != nil {
					fmt.Printf("in search test, couldn't parse rhs: %s - skipping", err.Error())
					continue
				}
				k = parsedRHS.NewIndex
				sk.Value = parsedRHS.Value

			case "default":
				rule.Kind.Family = KindFamilyDefault
			case "clear":
				rule.Kind.Family = KindFamilyClear
			case "name":
				rule.Kind.Family = KindFamilyName

				// eyy, new page
				page = string(test)
				ctx.Logf("now storing in page %s", page)
			case "use":
				uk := &UseKind{}
				rule.Kind.Family = KindFamilyUse
				rule.Kind.Data = uk

				k := 0
				if k+2 < len(test) && test[k] == '\\' && test[k+1] == '^' {
					k += 2
					uk.SwapEndian = true
				}

				uk.Page = string(test[k:])
			case "der":
				uk := &UseKind{}
				rule.Kind.Family = KindFamilyUse
				rule.Kind.Data = uk

				k := 0
				if k+2 < len(test) && test[k] == '\\' && test[k+1] == '^' {
					k += 2
					uk.SwapEndian = true
				}

				uk.Page = string(test[k:])
			default:
				ctx.Logf("unhandled kind (%s)\n", parsedKind.Value)
				continue
			}

			rule.Description = descriptionBytes
			book.AddRule(page, rule)
		}
	}

	return nil
}
