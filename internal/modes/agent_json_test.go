package modes

import "testing"

func TestParseGeneratedFilesJSON_Array(t *testing.T) {
	input := `[{"filename":"a.txt","content":"hello"},{"filename":"b.txt","content":"world"}]`
	files, err := ParseGeneratedFilesJSON(input)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	if files[0].Filename != "a.txt" || files[0].Content != "hello" {
		t.Fatalf("unexpected first file: %#v", files[0])
	}
}

func TestParseGeneratedFilesJSON_Object(t *testing.T) {
	input := `{"filename":"a.txt","content":"hello"}`
	files, err := ParseGeneratedFilesJSON(input)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Filename != "a.txt" || files[0].Content != "hello" {
		t.Fatalf("unexpected file: %#v", files[0])
	}
}

func TestParseGeneratedFilesJSON_Invalid(t *testing.T) {
	_, err := ParseGeneratedFilesJSON(`not json`)
	if err == nil {
		t.Fatalf("expected error")
	}
}
