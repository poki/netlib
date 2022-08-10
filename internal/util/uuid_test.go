package util_test

import (
	"testing"

	"github.com/poki/netlib/internal/util"
)

func Test_IsUUID(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"9c5c9b68-1a95-11ea-bd39-9cb6d0d995f7", true},
		{"0c5c9b68-1a95-11ea-bd39-9cb6d0d995f7", true},
		{"ac5c9b68-1a95-11ea-bd39-9cb6d0d995f7", true},
		{"fc5c9b68-1a95-11ea-bd39-9cb6d0d995f7", true},
		{"9c5c9b68-1a95-11ea-bd39-9cb6d0d995f9", true},
		{"9c5c9b68-1a95-11ea-bd39-9cb6d0d995f0", true},
		{"9c5c9b68-1a95-11ea-bd39-9cb6d0d995ff", true},
		{"9c5c9b68-1a95-11ea-bd39-9cb6d0d995fa", true},
		{"9C5C9B68-1A95-11EA-BD39-9CB6D0D995FA", true},
		{"9c5c9b68-1a95-11ea-bd39-9cb6d0d995a", false},
		{"9c5c9b68-1a95-11ea-bd39-9cb6dqd995fa", false},
		{"9c5c9b68-1a95-11ea-bd399-cb6dqd995fa", false},
		{"test", false},
		{"t-e-s-t-s", false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := util.IsUUID(tt.in); got != tt.want {
				t.Errorf("IsUUID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func FuzzIsUUID(f *testing.F) {
	f.Add("10e4dd45-ecff-4210-a43e-9bb1973f4dbc")
	f.Add("a1edb044-6a63-4cb4-be57-8d7c5764f70b")
	f.Add("A0FE65C6-0A3C-474B-B1CB-0A5E05D18754")
	f.Add("q336d067-2af0-4d81-a31a-41q876d0fd4b")
	f.Add("")
	f.Add("-")
	f.Add("🔥")
	f.Add("foobar")
	f.Fuzz(func(t *testing.T, data string) {
		util.IsUUID(data)
	})
}

var result any

func BenchmarkIsUUID(t *testing.B) {
	var r bool
	for i := 0; i < t.N; i++ {
		r = util.IsUUID("9c5c9b68-1a95-11ea-bd39-9cb6d0d995f7")
	}
	result = r
}
