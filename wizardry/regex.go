package wizardry

import (
	"bytes"
	"encoding/binary"
	"github.com/yizhibenpao/go-filetype/wizardry/wizutil"
	"regexp"
)

type RegexTestFlags int64

const (
	Case_insensitive = 1 << iota
	Match_To_Start
	Limit_Lines
	Trim
)

func IntToBytes(n int) []byte {
	x := int32(n)

	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}

// StringTest looks for a string pattern in target, at given index
func RegexTest(sr *wizutil.SliceReader, targetIndex int64, maxLen int64, regexString string, flags RegexTestFlags) int64 {
	bv := &wizutil.ByteView{
		Input:    sr,
		LookBack: 0,
	}

	if flags&Case_insensitive > 0 {
		regexString = "(?i)" + regexString
	}
	testp := regexp.MustCompile(regexString)
	if flags&Match_To_Start > 0 {
		targetIndex = 0
	}
	testbyte := []byte("")
	if flags&Limit_Lines > 0 {
		maxLen = maxLen * 80
		for i := int64(0); i < maxLen; i++ {
			if bv.Get(targetIndex+i) != 10 {
				testbyte = append(testbyte, IntToBytes(bv.Get(targetIndex+i))...)
			} else {
				if flags&Trim > 0 {
					testbyte = bytes.Trim(testbyte, " ")
				}
				loc := testp.FindIndex(testbyte)
				if loc == nil {
					testbyte = []byte("")
				} else {
					return int64(loc[1])
				}
			}
		}
		return -1
	} else {
		for i := int64(0); i < maxLen; i++ {
			testbyte = append(testbyte, IntToBytes(bv.Get(targetIndex+i))...)
		}
		if flags&Trim > 0 {
			testbyte = bytes.Trim(testbyte, " ")
		}
		loc := testp.FindIndex(testbyte)
		if loc == nil {
			return -1
		} else {
			return int64(loc[1])
		}
	}
	return -1
}
