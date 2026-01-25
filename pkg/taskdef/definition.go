package taskdef

import "encoding/json"

type Field struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
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
	Field  string
	Reason string
}

func (e *ParseError) Error() string {
	return "parse error: " + e.Reason + " for field " + e.Field
}

func ParseTaskInput(inputSchema []Field, params map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	for _, field := range inputSchema {
		value, ok := params[field.Name]
		if !ok {
			if field.Required {
				return nil, &ParseError{Field: field.Name, Reason: "required field missing"}
			}
			continue
		}
		result[field.Name] = value
	}
	return result, nil
}
