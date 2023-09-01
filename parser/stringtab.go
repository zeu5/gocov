package parser

// This package implements string table and reader utilities,
// for use in emitting and reading/decoding coverage meta-data and
// counter-data files.
// Reader is a helper for reading a string table previously
// serialized by a Writer.Write call.
type SReader struct {
	r    *Reader
	strs []string
}

// NewReader creates a stringtab.Reader to read the contents
// of a string table from 'r'.
func NewSReader(r *Reader) *SReader {
	str := &SReader{
		r: r,
	}
	return str
}

// Read reads/decodes a string table using the reader provided.
func (str *SReader) Read() {
	numEntries := int(str.r.ReadULEB128())
	str.strs = make([]string, 0, numEntries)
	for idx := 0; idx < numEntries; idx++ {
		slen := str.r.ReadULEB128()
		str.strs = append(str.strs, str.r.ReadString(int64(slen)))
	}
}

// Entries returns the number of decoded entries in a string table.
func (str *SReader) Entries() int {
	return len(str.strs)
}

// Get returns string 'idx' within the string table.
func (str *SReader) Get(idx uint32) string {
	return str.strs[idx]
}

func AppendUleb128(b []byte, v uint) []byte {
	for {
		c := uint8(v & 0x7f)
		v >>= 7
		if v != 0 {
			c |= 0x80
		}
		b = append(b, c)
		if c&0x80 == 0 {
			break
		}
	}
	return b
}
