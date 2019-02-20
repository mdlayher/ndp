package ndp

import "testing"

func Test_fuzz(t *testing.T) {
	tests := []struct {
		name string
		s    string
	}{
		{
			name: "parse option length",
			s:    "\x86000000000000000\x01\xc0",
		},
		{
			name: "prefix information length",
			s: "\x86000000000000000\x03\x0100" +
				"0000",
		},
		{
			name: "raw option marshal symmetry",
			s: "\x860000000000000000!00" +
				"00000000000000000000" +
				"00000000000000000000" +
				"00000000000000000000" +
				"00000000000000000000" +
				"00000000000000000000" +
				"00000000000000000000" +
				"00000000000000000000" +
				"00000000000000000000" +
				"00000000000000000000" +
				"00000000000000000000" +
				"00000000000000000000" +
				"00000000000000000000" +
				"00000000000000000000",
		},
		{
			name: "rdnss no servers",
			s:    "\x850000000\x19\x01000000",
		},
		{
			name: "dnssl bad domain",
			s: "\x850000000\x1f\x02000000\x02.0\x01" +
				"0\x00\x000",
		},
		{
			name: "dnssl length padding",
			s: "\x86000000000000000\x1f\b00" +
				"0000\x010\x010\x010\x1d000000000" +
				"00000000000000000000" +
				"\x010\x010\x00\x0000000000000000",
		},
		{
			name: "dnssl early termination no padding",
			s: "\x850000000\x1f\f000000\x010\x010" +
				"\x0200\x00\x00000000000000000" +
				"00000000000000000000" +
				"00000000000000000000" +
				"00000000000000000000" +
				"0000",
		},
		{
			name: "dnssl early termination one pad null",
			s: "\x850000000\x1f\a000000\x0200\x00" +
				"\t000000000\x00\x0000000000" +
				"00000000000000000000" +
				"0000",
		},
		{
			name: "dnssl punycode empty string",
			s: "\x850000000\x1f\x02000000\x04xn-" +
				"-\x00\x000",
		},
		{
			name: "dnssl with spaces",
			s: "\x850000000\x1f\x03000000\x05．" +
				"00\x010\x0500000\x00\x00",
		},
		{
			name: "dnssl not ASCII",
			s: "\x850000000\x1f\x02000000\x06．" +
				"000\x00",
		},
		{
			name: "dnssl decodes to empty string",
			s: "\x850000000\x1f\x02000000\x04xn-" +
				"-\x010\x00",
		},
		{
			name: "dnssl unicode replacement character",
			s: "\x850000000\x1f\x04000000\x010\x020" +
				"0\x0exn---00000H00F\x01@\x00\x00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = fuzz([]byte(tt.s))
		})
	}
}
