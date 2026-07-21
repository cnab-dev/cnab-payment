package itau240

import "testing"

func TestDecodeBatchHeader(t *testing.T) {
	tests := []struct {
		name    string
		raw     []byte
		want    batchHeader
		wantErr bool
	}{
		{"validBatchHeader_decodesBatchNumber", batchHeaderBytes("0001"), batchHeader{BatchNumber: "0001"}, false},
		{"wrongLength_returnsError", []byte("too short"), batchHeader{}, true},
		{"wrongRecordType_returnsError", detailBytes("0001", "00002", 'A', "", ""), batchHeader{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeBatchHeader(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("decodeBatchHeader() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Errorf("decodeBatchHeader() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestDecodeBatchTrailer(t *testing.T) {
	tests := []struct {
		name    string
		raw     []byte
		want    batchTrailer
		wantErr bool
	}{
		{"validBatchTrailer_decodesBatchNumber", batchTrailerBytes("0001"), batchTrailer{BatchNumber: "0001"}, false},
		{"wrongLength_returnsError", []byte("too short"), batchTrailer{}, true},
		{"wrongRecordType_returnsError", detailBytes("0001", "00002", 'A', "", ""), batchTrailer{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeBatchTrailer(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("decodeBatchTrailer() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Errorf("decodeBatchTrailer() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
