package masker

import (
	"testing"
)

func TestParseStrategy(t *testing.T) {
	tests := []struct {
		name        string
		strategy    string
		wantType    StrategyType
		wantFirst   int
		wantLast    int
		wantMaskSeq int
		wantKeepMid int
		wantErr     bool
	}{
		{
			name:      "simple mask-2-2",
			strategy:  "mask-2-2",
			wantType:  StrategyKeepFirstLast,
			wantFirst: 2,
			wantLast:  2,
			wantErr:   false,
		},
		{
			name:      "mask-0-0 (mask everything)",
			strategy:  "mask-0-0",
			wantType:  StrategyKeepFirstLast,
			wantFirst: 0,
			wantLast:  0,
			wantErr:   false,
		},
		{
			name:        "sequenced mask-2-2-5-1",
			strategy:    "mask-2-2-5-1",
			wantType:    StrategySequenced,
			wantFirst:   2,
			wantLast:    2,
			wantMaskSeq: 5,
			wantKeepMid: 1,
			wantErr:     false,
		},
		{
			name:     "invalid format - no mask prefix",
			strategy: "keep-2-2",
			wantErr:  true,
		},
		{
			name:     "invalid format - not enough numbers",
			strategy: "mask-2",
			wantErr:  true,
		},
		{
			name:     "invalid format - non-numeric",
			strategy: "mask-2-abc",
			wantErr:  true,
		},
		{
			name:     "invalid format - too many numbers",
			strategy: "mask-2-2-5-1-3",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseStrategy(tt.strategy)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseStrategy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Type != tt.wantType {
				t.Errorf("ParseStrategy() Type = %v, want %v", got.Type, tt.wantType)
			}
			if got.KeepFirst != tt.wantFirst {
				t.Errorf("ParseStrategy() KeepFirst = %v, want %v", got.KeepFirst, tt.wantFirst)
			}
			if got.KeepLast != tt.wantLast {
				t.Errorf("ParseStrategy() KeepLast = %v, want %v", got.KeepLast, tt.wantLast)
			}
			if tt.wantType == StrategySequenced {
				if got.MaskSeq != tt.wantMaskSeq {
					t.Errorf("ParseStrategy() MaskSeq = %v, want %v", got.MaskSeq, tt.wantMaskSeq)
				}
				if got.KeepMid != tt.wantKeepMid {
					t.Errorf("ParseStrategy() KeepMid = %v, want %v", got.KeepMid, tt.wantKeepMid)
				}
			}
		})
	}
}

func TestMaskKeepFirstLast(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		keepFirst int
		keepLast  int
		maskChar  string
		want      string
	}{
		{
			name:      "basic mask-2-2",
			value:     "secretpassword123",
			keepFirst: 2,
			keepLast:  2,
			maskChar:  "*",
			want:      "se*************23",
		},
		{
			name:      "mask-0-0 masks everything",
			value:     "secret",
			keepFirst: 0,
			keepLast:  0,
			maskChar:  "*",
			want:      "******",
		},
		{
			name:      "value too short - mask everything",
			value:     "abc",
			keepFirst: 2,
			keepLast:  2,
			maskChar:  "*",
			want:      "***",
		},
		{
			name:      "exact length - mask everything",
			value:     "abcd",
			keepFirst: 2,
			keepLast:  2,
			maskChar:  "*",
			want:      "****",
		},
		{
			name:      "custom mask char",
			value:     "password123",
			keepFirst: 2,
			keepLast:  3,
			maskChar:  "#",
			want:      "pa######123",
		},
		{
			name:      "keep more of start",
			value:     "postgres://user:pass@host:5432/db",
			keepFirst: 10,
			keepLast:  8,
			maskChar:  "*",
			want:      "postgres:/***************:5432/db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskKeepFirstLast(tt.value, tt.keepFirst, tt.keepLast, tt.maskChar)
			if got != tt.want {
				t.Errorf("MaskKeepFirstLast() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMaskSequenced(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		keepFirst int
		keepLast  int
		maskSeq   int
		keepMid   int
		maskChar  string
		want      string
	}{
		{
			name:      "basic sequenced pattern",
			value:     "1234567890abcdefgh",
			keepFirst: 2,
			keepLast:  2,
			maskSeq:   5,
			keepMid:   1,
			maskChar:  "*",
			want:      "12*****8*****d**gh",
		},
		{
			name:      "postgres connection string",
			value:     "postgres://user:secretpassword@localhost:5432/mydb",
			keepFirst: 12,
			keepLast:  15,
			maskSeq:   8,
			keepMid:   2,
			maskChar:  "*",
			want:      "postgres://u********et********@l***lhost:5432/mydb",
		},
		{
			name:      "short value - mask everything",
			value:     "abc",
			keepFirst: 2,
			keepLast:  2,
			maskSeq:   5,
			keepMid:   1,
			maskChar:  "*",
			want:      "***",
		},
		{
			name:      "middle exactly fits one sequence",
			value:     "ab12345cd",
			keepFirst: 2,
			keepLast:  2,
			maskSeq:   5,
			keepMid:   0,
			maskChar:  "*",
			want:      "ab*****cd",
		},
		{
			name:      "middle smaller than maskSeq - mask remainder",
			value:     "ab123cd",
			keepFirst: 2,
			keepLast:  2,
			maskSeq:   5,
			keepMid:   1,
			maskChar:  "*",
			want:      "ab***cd",
		},
		{
			name:      "MongoDB connection string",
			value:     "mongodb://admin:MySecretP@ssw0rd!@cluster0.mongodb.net/test?retryWrites=true",
			keepFirst: 15,
			keepLast:  40,
			maskSeq:   10,
			keepMid:   2,
			maskChar:  "*",
			want:      "mongodb://admin**********@s*********uster0.mongodb.net/test?retryWrites=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskSequenced(tt.value, tt.keepFirst, tt.keepLast, tt.maskSeq, tt.keepMid, tt.maskChar)
			if got != tt.want {
				t.Errorf("MaskSequenced()\n got: %v\nwant: %v", got, tt.want)
			}
		})
	}
}

func TestStrategyExecute(t *testing.T) {
	tests := []struct {
		name     string
		strategy Strategy
		value    string
		maskChar string
		want     string
	}{
		{
			name: "KeepFirstLast strategy",
			strategy: Strategy{
				Type:      StrategyKeepFirstLast,
				KeepFirst: 2,
				KeepLast:  2,
			},
			value:    "password123",
			maskChar: "*",
			want:     "pa*******23",
		},
		{
			name: "Sequenced strategy",
			strategy: Strategy{
				Type:      StrategySequenced,
				KeepFirst: 2,
				KeepLast:  2,
				MaskSeq:   3,
				KeepMid:   1,
			},
			value:    "1234567890",
			maskChar: "*",
			want:     "12***6**90",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.strategy.Execute(tt.value, tt.maskChar)
			if got != tt.want {
				t.Errorf("Strategy.Execute() = %v, want %v", got, tt.want)
			}
		})
	}
}
