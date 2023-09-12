package gocov

// This package implements string table and reader utilities,
// for use in emitting and reading/decoding coverage meta-data and
// counter-data files.
// Reader is a helper for reading a string table previously
// serialized by a Writer.Write call.
type sReader struct {
	r    *reader
	strs []string
}

// NewReader creates a stringtab.Reader to read the contents
// of a string table from 'r'.
func newSReader(r *reader) *sReader {
	str := &sReader{
		r: r,
	}
	return str
}

// Read reads/decodes a string table using the reader provided.
func (str *sReader) Read() {
	numEntries := int(str.r.ReadULEB128())
	str.strs = make([]string, 0, numEntries)
	for idx := 0; idx < numEntries; idx++ {
		slen := str.r.ReadULEB128()
		str.strs = append(str.strs, str.r.ReadString(int64(slen)))
	}
}

// Entries returns the number of decoded entries in a string table.
func (str *sReader) Entries() int {
	return len(str.strs)
}

// Get returns string 'idx' within the string table.
func (str *sReader) Get(idx uint32) string {
	return str.strs[idx]
}
