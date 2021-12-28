package auth

import (
	"os"
	"strings"
	"testing"
)

func TestBarcodeHTMLfile(t *testing.T) {
	type args struct {
		barcode string
		secret  string
	}
	tests := []struct {
		name       string
		args       args
		wantSecret string
		wantErr    bool
	}{
		{
			name: "Check secret in html",
			args: args{
				barcode: "foo",
				secret:  "SuperSecretCode",
			},
			wantSecret: "SuperSecretCode",
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := BarcodeHTMLfile(tt.args.barcode, tt.args.secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("BarcodeHTMLfile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			defer os.Remove(file.Name())

			b, err := os.ReadFile(file.Name())
			if err != nil {
				t.Fatalf("could not read temp file %s", err)
			}
			body := string(b)
			if !strings.Contains(body, tt.wantSecret) {
				t.Fatalf("expected secret string %s got none", tt.wantSecret)
			}
			if !strings.Contains(body, "qr-image") {
				t.Fatal("Expect img element with id qr-image, got none")
			}
		})
	}
}
