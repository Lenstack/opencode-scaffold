package template

import (
	"bytes"
	"testing"
	"text/template"
)

func TestFuncMap(t *testing.T) {
	funcs := FuncMap()

	if funcs["kebabcase"] == nil {
		t.Error("expected kebabcase function")
	}
	if funcs["snakecase"] == nil {
		t.Error("expected snakecase function")
	}
	if funcs["camelcase"] == nil {
		t.Error("expected camelcase function")
	}
	if funcs["titlecase"] == nil {
		t.Error("expected titlecase function")
	}
	if funcs["upper"] == nil {
		t.Error("expected upper function")
	}
	if funcs["lower"] == nil {
		t.Error("expected lower function")
	}
	if funcs["replace"] == nil {
		t.Error("expected replace function")
	}
	if funcs["trim"] == nil {
		t.Error("expected trim function")
	}
}

func TestFuncMap_Kebabcase(t *testing.T) {
	funcs := FuncMap()
	kebabcase := funcs["kebabcase"].(func(string) string)

	tests := []struct {
		input    string
		expected string
	}{
		{"helloWorld", "hello-world"},
		{"HelloWorld", "hello-world"},
		{"TestName", "test-name"},
		{"simple", "simple"},
	}

	for _, tt := range tests {
		result := kebabcase(tt.input)
		if result != tt.expected {
			t.Errorf("kebabcase(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestFuncMap_Snakecase(t *testing.T) {
	funcs := FuncMap()
	snakecase := funcs["snakecase"].(func(string) string)

	tests := []struct {
		input    string
		expected string
	}{
		{"helloWorld", "hello_world"},
		{"HelloWorld", "hello_world"},
		{"TestName", "test_name"},
		{"simple", "simple"},
	}

	for _, tt := range tests {
		result := snakecase(tt.input)
		if result != tt.expected {
			t.Errorf("snakecase(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestFuncMap_Camelcase(t *testing.T) {
	funcs := FuncMap()
	camelcase := funcs["camelcase"].(func(string) string)

	tests := []struct {
		input    string
		expected string
	}{
		{"hello_world", "HelloWorld"},
		{"test_name", "TestName"},
		{"simple", "Simple"},
	}

	for _, tt := range tests {
		result := camelcase(tt.input)
		if result != tt.expected {
			t.Errorf("camelcase(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestFuncMap_Titlecase(t *testing.T) {
	funcs := FuncMap()
	titlecase := funcs["titlecase"].(func(string) string)

	result := titlecase("hello world")
	if result != "Hello World" {
		t.Errorf("titlecase('hello world') = %q, want %q", result, "Hello World")
	}
}

func TestFuncMap_Upper(t *testing.T) {
	funcs := FuncMap()
	upper := funcs["upper"].(func(string) string)

	result := upper("hello")
	if result != "HELLO" {
		t.Errorf("upper('hello') = %q, want %q", result, "HELLO")
	}
}

func TestFuncMap_Lower(t *testing.T) {
	funcs := FuncMap()
	lower := funcs["lower"].(func(string) string)

	result := lower("HELLO")
	if result != "hello" {
		t.Errorf("lower('HELLO') = %q, want %q", result, "hello")
	}
}

func TestFuncMap_Replace(t *testing.T) {
	funcs := FuncMap()
	replace := funcs["replace"].(func(string, string, string) string)

	result := replace("hello world", "world", "go")
	if result != "hello go" {
		t.Errorf("replace('hello world', 'world', 'go') = %q, want %q", result, "hello go")
	}
}

func TestFuncMap_Trim(t *testing.T) {
	funcs := FuncMap()
	trim := funcs["trim"].(func(string) string)

	result := trim("  hello  ")
	if result != "hello" {
		t.Errorf("trim('  hello  ') = %q, want %q", result, "hello")
	}
}

func TestFuncMap_Integration(t *testing.T) {
	// Test that functions work in actual template
	tmpl := template.Must(template.New("test").Funcs(FuncMap()).Parse(`{{kebabcase "HelloWorld"}}`))

	var buf bytes.Buffer
	err := tmpl.Execute(&buf, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	result := buf.String()
	if result != "hello-world" {
		t.Errorf("template output = %q, want %q", result, "hello-world")
	}
}
