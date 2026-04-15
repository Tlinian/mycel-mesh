package encoding

import (
	"testing"
)

// TestJSONCodec_Marshal tests JSON marshaling.
func TestJSONCodec_Marshal(t *testing.T) {
	codec := jsonCodec{}

	data := map[string]string{"key": "value"}
	result, err := codec.Marshal(data)
	if err != nil {
		t.Fatalf("Marshal() failed: %v", err)
	}

	expected := `{"key":"value"}`
	if string(result) != expected {
		t.Fatalf("expected '%s', got '%s'", expected, string(result))
	}
}

// TestJSONCodec_Unmarshal tests JSON unmarshaling.
func TestJSONCodec_Unmarshal(t *testing.T) {
	codec := jsonCodec{}

	input := `{"key":"value"}`
	target := make(map[string]string)

	err := codec.Unmarshal([]byte(input), &target)
	if err != nil {
		t.Fatalf("Unmarshal() failed: %v", err)
	}

	if target["key"] != "value" {
		t.Fatalf("expected key='value', got '%s'", target["key"])
	}
}

// TestJSONCodec_Name tests codec name.
func TestJSONCodec_Name(t *testing.T) {
	codec := jsonCodec{}

	if codec.Name() != Name {
		t.Fatalf("expected Name '%s', got '%s'", Name, codec.Name())
	}
}

// TestJSONCodec_String tests codec string representation.
func TestJSONCodec_String(t *testing.T) {
	codec := jsonCodec{}

	if codec.String() != Name {
		t.Fatalf("expected String '%s', got '%s'", Name, codec.String())
	}
}

// TestJSONCodecWithLenPrefix_Marshal tests length-prefix codec marshaling.
func TestJSONCodecWithLenPrefix_Marshal(t *testing.T) {
	codec := jsonCodecWithLenPrefix{}

	data := map[string]int{"count": 42}
	result, err := codec.Marshal(data)
	if err != nil {
		t.Fatalf("Marshal() failed: %v", err)
	}

	expected := `{"count":42}`
	if string(result) != expected {
		t.Fatalf("expected '%s', got '%s'", expected, string(result))
	}
}

// TestJSONCodecWithLenPrefix_Unmarshal tests length-prefix codec unmarshaling.
func TestJSONCodecWithLenPrefix_Unmarshal(t *testing.T) {
	codec := jsonCodecWithLenPrefix{}

	input := `{"count":42}`
	target := make(map[string]int)

	err := codec.Unmarshal([]byte(input), &target)
	if err != nil {
		t.Fatalf("Unmarshal() failed: %v", err)
	}

	if target["count"] != 42 {
		t.Fatalf("expected count=42, got %d", target["count"])
	}
}

// TestJSONCodecWithLenPrefix_Name tests length-prefix codec name.
func TestJSONCodecWithLenPrefix_Name(t *testing.T) {
	codec := jsonCodecWithLenPrefix{}

	if codec.Name() != "proto" {
		t.Fatalf("expected Name 'proto', got '%s'", codec.Name())
	}
}

// TestJSONCodecWithLenPrefix_String tests length-prefix codec string.
func TestJSONCodecWithLenPrefix_String(t *testing.T) {
	codec := jsonCodecWithLenPrefix{}

	if codec.String() != "json-proto" {
		t.Fatalf("expected String 'json-proto', got '%s'", codec.String())
	}
}

// TestMarshalUnmarshalRoundtrip tests roundtrip of complex data.
func TestMarshalUnmarshalRoundtrip(t *testing.T) {
	codec := jsonCodec{}

	original := map[string]interface{}{
		"string":  "hello",
		"number":  123,
		"boolean": true,
		"array":   []int{1, 2, 3},
	}

	marshaled, err := codec.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() failed: %v", err)
	}

	target := make(map[string]interface{})
	err = codec.Unmarshal(marshaled, &target)
	if err != nil {
		t.Fatalf("Unmarshal() failed: %v", err)
	}

	// Verify key existence
	keys := []string{"string", "number", "boolean", "array"}
	for _, key := range keys {
		if _, exists := target[key]; !exists {
			t.Fatalf("key '%s' not found in unmarshaled data", key)
		}
	}
}

// TestMarshal_InvalidType tests marshaling invalid type.
func TestMarshal_InvalidType(t *testing.T) {
	codec := jsonCodec{}

	// Channels cannot be JSON marshaled
	_, err := codec.Marshal(make(chan int))
	if err == nil {
		t.Fatal("expected error for invalid type (channel)")
	}
}

// TestUnmarshal_InvalidJSON tests unmarshaling invalid JSON.
func TestJSONCodec_Unmarshal_InvalidJSON(t *testing.T) {
	codec := jsonCodec{}

	target := make(map[string]string)
	err := codec.Unmarshal([]byte("not valid json"), &target)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// TestUnmarshal_TypeMismatch tests unmarshaling with type mismatch.
func TestJSONCodec_Unmarshal_TypeMismatch(t *testing.T) {
	codec := jsonCodec{}

	// JSON has string, target expects int
	input := `{"number":"not-a-number"}`
	target := struct {
		Number int `json:"number"`
	}{}

	err := codec.Unmarshal([]byte(input), &target)
	if err == nil {
		t.Fatal("expected error for type mismatch")
	}
}