package itau240

import (
	"testing"

	"github.com/cnab-dev/cnab-payment/internal/core"
)

func TestDecodeDetail(t *testing.T) {
	tests := []struct {
		name    string
		rec     core.Record
		want    detail
		wantErr bool
	}{
		{
			"validDetail_decodesFields",
			core.Record{Raw: detailBytes("0001", "00002", 'A', "PAY-1", "02"), Line: 5},
			detail{BatchNumber: "0001", Sequence: "00002", Segment: 'A', Raw: detailBytes("0001", "00002", 'A', "PAY-1", "02"), Line: 5},
			false,
		},
		{"wrongLength_returnsError", core.Record{Raw: []byte("too short"), Line: 1}, detail{}, true},
		{"wrongRecordType_returnsError", core.Record{Raw: batchHeaderBytes("0001"), Line: 1}, detail{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeDetail(tt.rec)
			if (err != nil) != tt.wantErr {
				t.Fatalf("decodeDetail() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if got.BatchNumber != tt.want.BatchNumber || got.Sequence != tt.want.Sequence || got.Segment != tt.want.Segment || got.Line != tt.want.Line {
				t.Errorf("decodeDetail() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
