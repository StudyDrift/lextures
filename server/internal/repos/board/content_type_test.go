package board

import "testing"

func TestValidateCreatePost(t *testing.T) {
	t.Parallel()
	link := "https://example.com/article"
	att := "11111111-1111-1111-1111-111111111111"

	cases := []struct {
		name    string
		in      CreatePostInput
		wantErr bool
	}{
		{"text ok", CreatePostInput{ContentType: "text", Title: "Hi"}, false},
		{"text empty", CreatePostInput{ContentType: "text"}, true},
		{"link missing url", CreatePostInput{ContentType: "link"}, true},
		{"link ok", CreatePostInput{ContentType: "link", LinkURL: link}, false},
		{"image missing att", CreatePostInput{ContentType: "image"}, true},
		{"image ok", CreatePostInput{ContentType: "image", AttachmentID: &att}, false},
		{"drawing missing", CreatePostInput{ContentType: "drawing"}, true},
		{"drawing ok", CreatePostInput{ContentType: "drawing", DrawingData: []byte(`[]`)}, false},
		{"bad type", CreatePostInput{ContentType: "sticker", Title: "x"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCreatePost(tc.in)
			if tc.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected: %v", err)
			}
		})
	}
}

func TestValidContentType(t *testing.T) {
	t.Parallel()
	if !ValidContentType("video") {
		t.Fatal("video should be valid")
	}
	if ValidContentType("gif") {
		t.Fatal("gif should be invalid")
	}
}
