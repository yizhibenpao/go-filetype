package wizparser

import (
	"fmt"
	"github.com/yizhibenpao/go-filetype/wizardry"
	"github.com/yizhibenpao/go-filetype/wizardry/wizutil"
	"regexp"
	"strconv"
)

type parsedInt struct {
	Value    int64
	NewIndex int
}

type parsedUint struct {
	Value    uint64
	NewIndex int
}

func parseInt(input []byte, j int) (*parsedInt, error) {
	inputSize := len(input)

	startJ := j
	if j < inputSize && input[j] == '-' {
		j++
	}

	base := 10

	if (j+1 < inputSize) && input[j] == '0' && input[j+1] == 'x' {
		// hexadecimal
		base = 16
		j += 2
		startJ = j
		for j < inputSize && wizutil.IsHexNumber(input[j]) {
			j++
		}
	} else if j+1 < inputSize && input[j] == '0' && wizutil.IsOctalNumber(input[j+1]) {
		// octal
		base = 8
		j++
		startJ = j
		for j < inputSize && wizutil.IsOctalNumber(input[j]) {
			j++
		}
	} else {
		// decimal
		for j < inputSize && wizutil.IsNumber(input[j]) {
			j++
		}
	}

	value, err := strconv.ParseInt(string(input[startJ:j]), base, 64)
	if err != nil {
		return nil, err
	}

	return &parsedInt{
		Value:    value,
		NewIndex: j,
	}, nil
}

func parseUint(input []byte, j int) (*parsedUint, error) {
	inputSize := len(input)
	startJ := j
	base := 10

	if (j+1 < inputSize) && input[j] == '0' && input[j+1] == 'x' {
		// hexadecimal
		base = 16
		j += 2
		startJ = j
		for j < inputSize && wizutil.IsHexNumber(input[j]) {
			j++
		}
	} else if j+1 < inputSize && input[j] == '0' && wizutil.IsOctalNumber(input[j+1]) {
		// octal
		base = 8
		j++
		startJ = j
		for j < inputSize && wizutil.IsOctalNumber(input[j]) {
			j++
		}
	} else {
		// decimal
		for j < inputSize && wizutil.IsNumber(input[j]) {
			j++
		}
	}

	value, err := strconv.ParseUint(string(input[startJ:j]), base, 64)
	if err != nil {
		return nil, err
	}

	return &parsedUint{
		Value:    value,
		NewIndex: j,
	}, nil
}

type parsedKind struct {
	Value    string
	NewIndex int
}

func parseKind(input []byte, j int) *parsedKind {
	inputSize := len(input)
	startJ := j

	for j < inputSize && (wizutil.IsNumber(input[j]) || wizutil.IsLowerLetter(input[j])) {
		j++
	}

	return &parsedKind{
		Value:    string(input[startJ:j]),
		NewIndex: j,
	}
}

type parsedString struct {
	Value    []byte
	NewIndex int
}

func parseString(input []byte, j int) (*parsedString, error) {
	inputSize := len(input)

	var result []byte
	for j < inputSize {
		if input[j] == '\\' {
			j++
			switch input[j] {
			case '\\':
				result = append(result, '\\')
				j++
			case 'r':
				result = append(result, '\r')
				j++
			case 'n':
				result = append(result, '\n')
				j++
			case 't':
				result = append(result, '\t')
				j++
			case 'v':
				result = append(result, '\v')
				j++
			case 'b':
				result = append(result, '\b')
				j++
			case 'a':
				result = append(result, '\a')
				j++
			case ' ':
				result = append(result, ' ')
				j++
			case 'x':
				j++
				// hexadecimal escape, e.g. "\x" or "\xeb"
				hexLen := 0
				if j < inputSize && wizutil.IsHexNumber(input[j]) {
					hexLen++
					if j+1 < inputSize && wizutil.IsHexNumber(input[j+1]) {
						hexLen++
					}
				}

				if hexLen == 0 {
					return nil, fmt.Errorf("invalid/unfinished hex escape in %s", input)
				}

				hexInput := string(input[j : j+hexLen])

				val, err := strconv.ParseUint(hexInput, 16, 8)
				if err != nil {
					return nil, fmt.Errorf("in hex escape %s: %s", hexInput, err.Error())
				}
				result = append(result, byte(val))
				j += hexLen
			case '<':
				val, _ := strconv.ParseUint("3C", 16, 8)
				result = append(result, byte(val))
				j++
			case '=':
				val, _ := strconv.ParseUint("3D", 16, 8)
				result = append(result, byte(val))
				j++
			case '!':
				val, _ := strconv.ParseUint("21", 16, 8)
				result = append(result, byte(val))
				j++
			case '&':
				val, _ := strconv.ParseUint("26", 16, 8)
				result = append(result, byte(val))
				j++
			case '>':
				val, _ := strconv.ParseUint("3E", 16, 8)
				result = append(result, byte(val))
				j++
			case 'f':
				val, _ := strconv.ParseUint("66", 16, 8)
				result = append(result, byte(val))
				j++
			default:
				if wizutil.IsOctalNumber(input[j]) {
					numOctal := 1
					k := j + 1
					for k < inputSize && numOctal < 3 && wizutil.IsOctalNumber(input[k]) {
						numOctal++
						k++
					}

					// octal escape e.g. "\0", "\11", "\222", but no longer
					octInput := string(input[j:k])
					val, err := strconv.ParseUint(octInput, 8, 8)
					if err != nil {
						return nil, fmt.Errorf("in oct escape %s: %s", octInput, err.Error())
					}
					result = append(result, byte(val))
					j = k
				} else {
					return nil, fmt.Errorf("unrecognized escape sequence starting with 0x%x, aka '\\%c'", input[j], input[j])
				}
			}
		} else {
			result = append(result, input[j])
			j++
		}
	}

	return &parsedString{
		Value:    result,
		NewIndex: j,
	}, nil
}

type parsedRegex struct {
	Value    []byte
	NewIndex int
}

func parseRegex(input []byte, j int) (*parsedRegex, error) {
	inputSize := len(input)

	var result []byte
	for j < inputSize {
		if input[j] == '\\' {
			j++
		}
		result = append(result, input[j])
		j++
	}

	return &parsedRegex{
		Value:    result,
		NewIndex: j,
	}, nil
}

type parsedStringTestFlags struct {
	Flags    wizardry.StringTestFlags
	NewIndex int
}

func parseStringTestFlags(input []byte, j int) *parsedStringTestFlags {
	inputSize := len(input)

	result := &parsedStringTestFlags{}

	for j < inputSize {
		switch input[j] {
		case 'W':
			result.Flags |= wizardry.CompactWhitespace
		case 'w':
			result.Flags |= wizardry.OptionalBlanks
		case 'c':
			result.Flags |= wizardry.LowerMatchesBoth
		case 'C':
			result.Flags |= wizardry.UpperMatchesBoth
		case 't':
			result.Flags |= wizardry.ForceText
		case 'b':
			result.Flags |= wizardry.ForceBinary
		default:
			break
		}
		j++
	}

	return result
}

type parsedRegexTestFlags struct {
	Flags    wizardry.RegexTestFlags
	Maxlen   int64
	NewIndex int
}

var regexflag = regexp.MustCompile("^regex(/(?P<length>\\d+)?(?P<flags>[cslT]*)(b\\d*)?)?$")

func parseRegexTestFlags(input []byte, j int) *parsedRegexTestFlags {

	result := &parsedRegexTestFlags{}

	flags := regexflag.FindAllStringSubmatch(string(input), -1)
	groupNames := regexflag.SubexpNames()

	for _, flag := range flags {
		for j, name := range groupNames {
			switch name {
			case "length":
				result.Maxlen, _ = strconv.ParseInt(flag[j], 10, 64)
			case "flags":
				i := 0
				for i < len(flag[j]) {
					switch flag[j][i] {
					case 'c':
						result.Flags |= wizardry.Case_insensitive
					case 's':
						result.Flags |= wizardry.Match_To_Start
					case 'l':
						result.Flags |= wizardry.Limit_Lines
					case 'T':
						result.Flags |= wizardry.Trim
					default:
						break
					}
					i++
				}
			}
		}

	}

	return result
}
