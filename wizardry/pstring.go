package wizardry

import "github.com/yizhibenpao/go-filetype/wizardry/wizutil"

// pstringTest looks for a fixed pattern at any position within a certain length
func PStringTest(sr *wizutil.SliceReader, targetIndex int64, LengthByte int, pattern string, EndiannessBIG bool, ContainsOwnLength bool) int64 {
	sf := MakeStringFinder(pattern)
	bv := &wizutil.ByteView{
		Input:    sr,
		LookBack: 0,
	}
	maxLen := int64(16)
	switch LengthByte {
	case 1:
		maxLen = int64(bv.Get(targetIndex))
	case 2:
		if EndiannessBIG {
			maxLen = int64(bv.Get(targetIndex)*16 + bv.Get(targetIndex+1))
		} else {
			maxLen = int64(bv.Get(targetIndex+1)*16 + bv.Get(targetIndex))
		}
	case 4:
		if EndiannessBIG {
			maxLen = int64(bv.Get(targetIndex)*4096 + bv.Get(targetIndex+1)*256 + bv.Get(targetIndex+2)*16 + bv.Get(targetIndex+3))
		} else {
			maxLen = int64(bv.Get(targetIndex+3)*4096 + bv.Get(targetIndex+2)*256 + bv.Get(targetIndex+1)*16 + bv.Get(targetIndex))
		}
	}
	if ContainsOwnLength {
		maxLen = maxLen - int64(LengthByte)
	}
	targetIndex = targetIndex + int64(LengthByte)
	sr = sr.Slice(targetIndex).Cap(maxLen)
	return sf.next(sr)
}
