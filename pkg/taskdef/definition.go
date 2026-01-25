package taskdef

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"time"
)

type Field struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Required bool        `json:"required"`
	Default  interface{} `json:"default,omitempty"`
}

type TaskConfig struct {
	Service string            `json:"service,omitempty"`
	Method  string            `json:"method,omitempty"`
	Topic   string            `json:"topic,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type TaskDefinition struct {
	Name        string     `json:"name"`
	Type        string     `json:"type"`
	Description string     `json:"description"`
	InputFields []Field    `json:"input_fields"`
	Config      TaskConfig `json:"config"`
}

var TaskDefinitions = map[string]TaskDefinition{
	"deduct": {
		Name:        "扣款",
		Type:        "rpc",
		Description: "从用户账户扣款",
		InputFields: []Field{
			{Name: "user_id", Type: "string", Required: true},
			{Name: "amount", Type: "int", Required: true},
			{Name: "order_id", Type: "string", Required: true},
		},
		Config: TaskConfig{
			Service: "PaymentService",
			Method:  "deduct",
		},
	},
	"notify": {
		Name:        "发送通知",
		Type:        "mq",
		Description: "发送支付完成通知",
		InputFields: []Field{
			{Name: "user_id", Type: "string", Required: true},
			{Name: "order_id", Type: "string", Required: true},
			{Name: "status", Type: "string", Required: true},
		},
		Config: TaskConfig{
			Topic: "payment.completed",
		},
	},
	"http_request": {
		Name:        "HTTP 请求",
		Type:        "http",
		Description: "发起 HTTP 请求",
		InputFields: []Field{
			{Name: "body", Type: "string", Required: false},
		},
		Config: TaskConfig{},
	},
}

func GetTaskDefinition(name string) (*TaskDefinition, error) {
	def, ok := TaskDefinitions[name]
	if !ok {
		return nil, nil
	}
	return &def, nil
}

func (t *TaskDefinition) GetInputFieldsJSON() string {
	data, _ := json.Marshal(t.InputFields)
	return string(data)
}

type ParseError struct {
	Field    string
	Reason   string
	GotType  string
	Expected string
}

func (e *ParseError) Error() string {
	if e.GotType != "" {
		return fmt.Sprintf("parse error: %s for field '%s', got %s, expected %s", e.Reason, e.Field, e.GotType, e.Expected)
	}
	return fmt.Sprintf("parse error: %s for field '%s'", e.Reason, e.Field)
}

type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

func (v *Validator) Validate(inputSchema []Field, params map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for _, field := range inputSchema {
		value, ok := params[field.Name]

		if !ok || value == nil {
			if field.Default != nil {
				result[field.Name] = field.Default
				continue
			}
			if field.Required {
				return nil, &ParseError{Field: field.Name, Reason: "required field missing"}
			}
			continue
		}

		converted, err := v.Convert(field, value)
		if err != nil {
			return nil, err
		}
		result[field.Name] = converted
	}

	return result, nil
}

func (v *Validator) Convert(field Field, value interface{}) (interface{}, error) {
	expectedType := field.Type

	switch expectedType {
	case "string":
		return v.toString(value)
	case "int":
		return v.toInt(value)
	case "int64":
		return v.toInt64(value)
	case "float":
		return v.toFloat(value)
	case "float64":
		return v.toFloat64(value)
	case "bool":
		return v.toBool(value)
	case "[]string":
		return v.toStringSlice(value)
	case "[]int":
		return v.toIntSlice(value)
	case "[]float":
		return v.toFloatSlice(value)
	case "object":
		return v.toObject(value)
	case "array":
		return v.toArray(value)
	case "time":
		return v.toTime(value)
	case "*":
		return value, nil
	default:
		return value, nil
	}
}

func (v *Validator) toString(value interface{}) (string, error) {
	switch val := value.(type) {
	case string:
		return val, nil
	case int:
		return strconv.Itoa(val), nil
	case int64:
		return strconv.FormatInt(val, 10), nil
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(val), nil
	case nil:
		return "", nil
	default:
		return fmt.Sprintf("%v", val), nil
	}
}

func (v *Validator) toInt(value interface{}) (int, error) {
	switch val := value.(type) {
	case int:
		return val, nil
	case int64:
		return int(val), nil
	case float64:
		return int(val), nil
	case string:
		i, err := strconv.Atoi(val)
		if err != nil {
			return 0, &ParseError{Reason: "invalid string format for int", GotType: "string", Expected: "int"}
		}
		return i, nil
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil
	case nil:
		return 0, nil
	default:
		return 0, &ParseError{Reason: "cannot convert to int", GotType: reflect.TypeOf(value).String(), Expected: "int"}
	}
}

func (v *Validator) toInt64(value interface{}) (int64, error) {
	switch val := value.(type) {
	case int:
		return int64(val), nil
	case int64:
		return val, nil
	case float64:
		return int64(val), nil
	case string:
		i, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return 0, &ParseError{Reason: "invalid string format for int64", GotType: "string", Expected: "int64"}
		}
		return i, nil
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil
	case nil:
		return 0, nil
	default:
		return 0, &ParseError{Reason: "cannot convert to int64", GotType: reflect.TypeOf(value).String(), Expected: "int64"}
	}
}

func (v *Validator) toFloat(value interface{}) (float32, error) {
	f, err := v.toFloat64(value)
	if err != nil {
		return 0, err
	}
	return float32(f), nil
}

func (v *Validator) toFloat64(value interface{}) (float64, error) {
	switch val := value.(type) {
	case float64:
		return val, nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, &ParseError{Reason: "invalid string format for float", GotType: "string", Expected: "float"}
		}
		return f, nil
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil
	case nil:
		return 0, nil
	default:
		return 0, &ParseError{Reason: "cannot convert to float", GotType: reflect.TypeOf(value).String(), Expected: "float"}
	}
}

func (v *Validator) toBool(value interface{}) (bool, error) {
	switch val := value.(type) {
	case bool:
		return val, nil
	case int:
		return val != 0, nil
	case int64:
		return val != 0, nil
	case string:
		b, err := strconv.ParseBool(val)
		if err != nil {
			return false, &ParseError{Reason: "invalid string format for bool", GotType: "string", Expected: "bool"}
		}
		return b, nil
	case float64:
		return val != 0, nil
	case nil:
		return false, nil
	default:
		return false, &ParseError{Reason: "cannot convert to bool", GotType: reflect.TypeOf(value).String(), Expected: "bool"}
	}
}

func (v *Validator) toStringSlice(value interface{}) ([]string, error) {
	switch val := value.(type) {
	case []string:
		return val, nil
	case []interface{}:
		result := make([]string, 0, len(val))
		for _, item := range val {
			s, err := v.toString(item)
			if err != nil {
				return nil, err
			}
			result = append(result, s)
		}
		return result, nil
	case string:
		if val == "" {
			return []string{}, nil
		}
		return []string{val}, nil
	case nil:
		return []string{}, nil
	default:
		return nil, &ParseError{Reason: "cannot convert to []string", GotType: reflect.TypeOf(value).String(), Expected: "[]string"}
	}
}

func (v *Validator) toIntSlice(value interface{}) ([]int, error) {
	switch val := value.(type) {
	case []int:
		return val, nil
	case []interface{}:
		result := make([]int, 0, len(val))
		for _, item := range val {
			i, err := v.toInt(item)
			if err != nil {
				return nil, err
			}
			result = append(result, i)
		}
		return result, nil
	case string:
		if val == "" {
			return []int{}, nil
		}
		i, err := v.toInt(val)
		if err != nil {
			return nil, err
		}
		return []int{i}, nil
	case nil:
		return []int{}, nil
	default:
		return nil, &ParseError{Reason: "cannot convert to []int", GotType: reflect.TypeOf(value).String(), Expected: "[]int"}
	}
}

func (v *Validator) toFloatSlice(value interface{}) ([]float64, error) {
	switch val := value.(type) {
	case []float64:
		return val, nil
	case []interface{}:
		result := make([]float64, 0, len(val))
		for _, item := range val {
			f, err := v.toFloat64(item)
			if err != nil {
				return nil, err
			}
			result = append(result, f)
		}
		return result, nil
	case string:
		if val == "" {
			return []float64{}, nil
		}
		f, err := v.toFloat64(val)
		if err != nil {
			return nil, err
		}
		return []float64{f}, nil
	case nil:
		return []float64{}, nil
	default:
		return nil, &ParseError{Reason: "cannot convert to []float", GotType: reflect.TypeOf(value).String(), Expected: "[]float"}
	}
}

func (v *Validator) toObject(value interface{}) (map[string]interface{}, error) {
	switch val := value.(type) {
	case map[string]interface{}:
		return val, nil
	case map[string]string:
		result := make(map[string]interface{})
		for k, v := range val {
			result[k] = v
		}
		return result, nil
	case string:
		if val == "" {
			return make(map[string]interface{}), nil
		}
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(val), &result); err != nil {
			return nil, &ParseError{Reason: "invalid JSON format for object", GotType: "string", Expected: "object"}
		}
		return result, nil
	case nil:
		return make(map[string]interface{}), nil
	default:
		return nil, &ParseError{Reason: "cannot convert to object", GotType: reflect.TypeOf(value).String(), Expected: "object"}
	}
}

func (v *Validator) toArray(value interface{}) ([]interface{}, error) {
	switch val := value.(type) {
	case []interface{}:
		return val, nil
	case []string:
		result := make([]interface{}, len(val))
		for i, v := range val {
			result[i] = v
		}
		return result, nil
	case []int:
		result := make([]interface{}, len(val))
		for i, v := range val {
			result[i] = v
		}
		return result, nil
	case string:
		if val == "" {
			return []interface{}{}, nil
		}
		var result []interface{}
		if err := json.Unmarshal([]byte(val), &result); err != nil {
			return nil, &ParseError{Reason: "invalid JSON format for array", GotType: "string", Expected: "array"}
		}
		return result, nil
	case nil:
		return []interface{}{}, nil
	default:
		return nil, &ParseError{Reason: "cannot convert to array", GotType: reflect.TypeOf(value).String(), Expected: "array"}
	}
}

func (v *Validator) toTime(value interface{}) (time.Time, error) {
	switch val := value.(type) {
	case time.Time:
		return val, nil
	case string:
		formats := []string{
			time.RFC3339,
			"2006-01-02T15:04:05Z",
			"2006-01-02 15:04:05",
			"2006-01-02",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, val); err == nil {
				return t, nil
			}
		}
		return time.Time{}, &ParseError{Reason: "invalid time format", GotType: "string", Expected: "time"}
	case int64:
		return time.Unix(val, 0), nil
	case nil:
		return time.Time{}, nil
	default:
		return time.Time{}, &ParseError{Reason: "cannot convert to time", GotType: reflect.TypeOf(value).String(), Expected: "time"}
	}
}

func ParseTaskInput(inputSchema []Field, params map[string]interface{}) (map[string]interface{}, error) {
	validator := NewValidator()
	return validator.Validate(inputSchema, params)
}

type ValidationResult struct {
	Valid  bool
	Data   map[string]interface{}
	Errors []ParseError
}

func ValidateAll(definitions map[string]TaskDefinition, taskName string, params map[string]interface{}) (*ValidationResult, error) {
	def, ok := definitions[taskName]
	if !ok {
		return nil, fmt.Errorf("task definition not found: %s", taskName)
	}

	if len(def.InputFields) == 0 {
		return &ValidationResult{Valid: true, Data: params}, nil
	}

	validator := NewValidator()
	data, err := validator.Validate(def.InputFields, params)
	if err != nil {
		if parseErr, ok := err.(*ParseError); ok {
			return &ValidationResult{
				Valid:  false,
				Errors: []ParseError{*parseErr},
			}, nil
		}
		return nil, err
	}

	return &ValidationResult{Valid: true, Data: data}, nil
}
