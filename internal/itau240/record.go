package itau240

// recordLength is the fixed length, in bytes, of every CNAB240 record.
const recordLength = 240

// Record type byte, found at column 8 (index 7) of every record.
const (
	recordTypeFileHeader   byte = '0'
	recordTypeBatchHeader  byte = '1'
	recordTypeDetail       byte = '3'
	recordTypeBatchTrailer byte = '5'
	recordTypeFileTrailer  byte = '9'
)

// File kind byte, found at column 143 (index 142) of the file header only.
const (
	fileKindRemessa byte = '1'
	fileKindRetorno byte = '2'
)

// bankCode is Itaú Unibanco's compensation code, found at columns 1-3
// (index 0:3) of every record.
const bankCode = "341"

// recordType returns the record-type byte (column 8) of raw, which must
// already be known to be recordLength bytes long.
func recordType(raw []byte) byte {
	return raw[7]
}

// batchNumber returns the "lote de serviço" field (columns 4-7), shared by
// every record type, used to correlate a batch's header, detail records,
// and trailer.
func batchNumber(raw []byte) string {
	return string(raw[3:7])
}
