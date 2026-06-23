package provider

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestBuildUserData(t *testing.T) {
	t.Parallel()

	binaryContent := []byte{0x1f, 0x8b, 0x08, 0x00, 0xff, 0xfe} // invalid UTF-8 (e.g. gzip header)

	tests := []struct {
		name         string
		content      []byte
		wantData     string
		wantIsBase64 bool
		wantErr      bool
	}{
		{
			name:         "plain text cloud-config is sent verbatim",
			content:      []byte("#cloud-config\nhostname: example\n"),
			wantData:     "#cloud-config\nhostname: example\n",
			wantIsBase64: false,
		},
		{
			name:         "empty content is valid UTF-8 and sent as is",
			content:      []byte(""),
			wantData:     "",
			wantIsBase64: false,
		},
		{
			name:         "multibyte UTF-8 is sent as is",
			content:      []byte("#cloud-config\n# café ☕\n"),
			wantData:     "#cloud-config\n# café ☕\n",
			wantIsBase64: false,
		},
		{
			name:         "non-UTF-8 binary content is base64-encoded",
			content:      binaryContent,
			wantData:     base64.StdEncoding.EncodeToString(binaryContent),
			wantIsBase64: true,
		},
		{
			name:    "text content exceeding the limit is rejected",
			content: []byte(strings.Repeat("a", flexvmUserDataMaxLen+1)),
			wantErr: true,
		},
		{
			name:    "binary content whose base64 exceeds the limit is rejected",
			content: append([]byte{0xff}, []byte(strings.Repeat("a", flexvmUserDataMaxLen))...),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := buildUserData(tt.content)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Data != tt.wantData {
				t.Errorf("Data = %q, want %q", got.Data, tt.wantData)
			}
			if got.IsBase64 != tt.wantIsBase64 {
				t.Errorf("IsBase64 = %v, want %v", got.IsBase64, tt.wantIsBase64)
			}
		})
	}
}
