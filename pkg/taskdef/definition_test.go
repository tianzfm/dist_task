package taskdef

import (
	"reflect"
	"testing"
)

func TestValidator_ToString(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name     string
		input    interface{}
		expected string
		wantErr  bool
	}{
		{"string", "hello", "hello", false},
		{"int to string", 123, "123", false},
		{"float to string", 123.45, "123.45", false},
		{"bool to string", true, "true", false},
		{"nil", nil, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := v.toString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("toString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("toString() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestValidator_ToInt(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name     string
		input    interface{}
		expected int
		wantErr  bool
	}{
		{"int", 123, 123, false},
		{"string int", "456", 456, false},
		{"bool true", true, 1, false},
		{"bool false", false, 0, false},
		{"nil", nil, 0, false},
		{"invalid string", "abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := v.toInt(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("toInt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("toInt() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestValidator_ToBool(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name     string
		input    interface{}
		expected bool
		wantErr  bool
	}{
		{"bool true", true, true, false},
		{"bool false", false, false, false},
		{"int 1", 1, true, false},
		{"int 0", 0, false, false},
		{"string true", "true", true, false},
		{"string false", "false", false, false},
		{"nil", nil, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := v.toBool(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("toBool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("toBool() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestValidator_Validate(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name        string
		inputSchema []Field
		params      map[string]interface{}
		wantData    map[string]interface{}
		wantErr     bool
	}{
		{
			name: "valid params",
			inputSchema: []Field{
				{Name: "user_id", Type: "string", Required: true},
				{Name: "amount", Type: "int", Required: true},
			},
			params: map[string]interface{}{
				"user_id": "12345",
				"amount":  100,
			},
			wantData: map[string]interface{}{
				"user_id": "12345",
				"amount":  100,
			},
			wantErr: false,
		},
		{
			name: "missing required field",
			inputSchema: []Field{
				{Name: "user_id", Type: "string", Required: true},
			},
			params:   map[string]interface{}{},
			wantData: nil,
			wantErr:  true,
		},
		{
			name: "type conversion",
			inputSchema: []Field{
				{Name: "amount", Type: "int", Required: true},
			},
			params: map[string]interface{}{
				"amount": "123",
			},
			wantData: map[string]interface{}{
				"amount": 123,
			},
			wantErr: false,
		},
		{
			name: "optional field missing",
			inputSchema: []Field{
				{Name: "user_id", Type: "string", Required: false},
			},
			params:   map[string]interface{}{},
			wantData: map[string]interface{}{},
			wantErr:  false,
		},
		{
			name: "invalid type",
			inputSchema: []Field{
				{Name: "amount", Type: "int", Required: true},
			},
			params: map[string]interface{}{
				"amount": "abc",
			},
			wantData: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := v.Validate(tt.inputSchema, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(result, tt.wantData) {
				t.Errorf("Validate() = %v, expected %v", result, tt.wantData)
			}
		})
	}
}

func TestValidator_ToStringSlice(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name     string
		input    interface{}
		expected []string
		wantErr  bool
	}{
		{"[]string", []string{"a", "b", "c"}, []string{"a", "b", "c"}, false},
		{"[]interface{}", []interface{}{"a", "b", "c"}, []string{"a", "b", "c"}, false},
		{"string single", "hello", []string{"hello"}, false},
		{"nil", nil, []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := v.toStringSlice(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("toStringSlice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("toStringSlice() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestValidator_ToObject(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name     string
		input    interface{}
		expected map[string]interface{}
		wantErr  bool
	}{
		{"map", map[string]interface{}{"key": "value"}, map[string]interface{}{"key": "value"}, false},
		{"JSON string", `{"key":"value"}`, map[string]interface{}{"key": "value"}, false},
		{"nil", nil, map[string]interface{}{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := v.toObject(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("toObject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("toObject() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestValidateAll(t *testing.T) {
	definitions := map[string]TaskDefinition{
		"deduct": {
			Name: "扣款",
			Type: "rpc",
			InputFields: []Field{
				{Name: "user_id", Type: "string", Required: true},
				{Name: "amount", Type: "int", Required: true},
			},
		},
	}

	tests := []struct {
		name      string
		taskName  string
		params    map[string]interface{}
		wantValid bool
		wantErr   bool
	}{
		{"valid", "deduct", map[string]interface{}{
			"user_id": "12345",
			"amount":  100,
		}, true, false},
		{"missing required", "deduct", map[string]interface{}{
			"user_id": "12345",
		}, false, false},
		{"task not found", "unknown", map[string]interface{}{}, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateAll(definitions, tt.taskName, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if result.Valid != tt.wantValid {
				t.Errorf("ValidateAll() valid = %v, wantValid %v", result.Valid, tt.wantValid)
			}
		})
	}
}
