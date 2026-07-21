package itau240

import "testing"

func TestDecodeFileTrailer(t *testing.T) {
	tests := []struct {
		name    string
		raw     []byte
		wantErr bool
	}{
		{"validFileTrailer_decodesWithoutError", fileTrailerBytes(), false},
		{"wrongLength_returnsError", []byte("too short"), true},
		{"wrongRecordType_returnsError", batchHeaderBytes("0001"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := decodeFileTrailer(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("decodeFileTrailer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
