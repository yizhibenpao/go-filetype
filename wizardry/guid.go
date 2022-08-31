package wizardry

import (
	"github.com/google/uuid"
	"github.com/yizhibenpao/go-filetype/wizardry/wizutil"
	"strings"
)

// StringTest looks for a string pattern in target, at given index
func GUIDTest(sr *wizutil.SliceReader, targetIndex int64, patternUUIDString string) int64 {
	bv := &wizutil.ByteView{
		Input:    sr,
		LookBack: 0,
	}

	pattern := []byte(patternUUIDString)
	patternSize := len(pattern)
	patternIndex := 0
	//UUID('{12345678-1234-5678-1234-567812345678}')
	//UUID('12345678123456781234567812345678')
	//UUID('urn:uuid:12345678-1234-5678-1234-567812345678')
	//UUID(bytes='\x12\x34\x56\x78'*4)
	//UUID(bytes_le='\x78\x56\x34\x12\x34\x12\x78\x56' +
	//	'\x12\x34\x56\x78\x12\x34\x56\x78')
	//UUID(fields=(0x12345678, 0x1234, 0x5678, 0x12, 0x34, 0x567812345678))
	//UUID(int=0x12345678123456781234567812345678)
	uuidtest := uuid.New()

	uuidtest.UnmarshalBinary([]byte{byte(bv.Get(targetIndex + 3)), byte(bv.Get(targetIndex + 2)), byte(bv.Get(targetIndex + 1)),
		byte(bv.Get(targetIndex + 0)), byte(bv.Get(targetIndex + 5)), byte(bv.Get(targetIndex + 4)), byte(bv.Get(targetIndex + 7)),
		byte(bv.Get(targetIndex + 6)), byte(bv.Get(targetIndex + 8)), byte(bv.Get(targetIndex + 9)), byte(bv.Get(targetIndex + 10)),
		byte(bv.Get(targetIndex + 11)), byte(bv.Get(targetIndex + 12)), byte(bv.Get(targetIndex + 13)), byte(bv.Get(targetIndex + 14)),
		byte(bv.Get(targetIndex + 15))})
	uuidbyte := []byte(strings.ToUpper(uuidtest.String()))

	for _, testuuid := range uuidbyte {
		patternByte := pattern[patternIndex]

		matches := patternByte == testuuid
		if matches {
			// perfect match, advance both
			patternIndex++
		} else {
			// not a match
			return -1
		}

		if patternIndex >= patternSize {
			// hey it matched all the way!
			return targetIndex + 16
		}
	}
	return -1
}
