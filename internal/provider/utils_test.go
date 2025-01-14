// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// utils_test.go
package provider

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/attr"
)

type TestStruct struct {
	Name       string  `json:"name"`
	Age        int     `json:"age"`
	IsEmployed bool    `json:"is_employed"`
	Address    *string `json:"address,omitempty"`
}

func TestJsonStructToObject(t *testing.T) {
	ctx := context.Background()
	address := "123 Main St"
	tests := []struct {
		name         string
		input        interface{}
		makeZeroNil  bool
		ignoreZero   bool
		expected     map[string]interface{}
		expectError  bool
	}{
		{
			name:        "valid struct with all fields",
			input:       &TestStruct{Name: "Alice", Age: 25, IsEmployed: true, Address: &address},
			makeZeroNil: false,
			ignoreZero:  false,
			expected: map[string]interface{}{
				"name":        "Alice",
				"age":         25,
				"is_employed": true,
				"address":     address,
			},
			expectError: false,
		},
		{
			name:        "ignore zero fields",
			input:       &TestStruct{Name: "Bob", Age: 0, IsEmployed: false, Address: nil},
			makeZeroNil: false,
			ignoreZero:  true,
			expected: map[string]interface{}{
				"name": "Bob",
			},
			expectError: false,
		},
		{
			name:        "error when input is not a struct",
			input:       "not a struct",
			makeZeroNil: false,
			ignoreZero:  false,
			expected:    nil,
			expectError: true,
		},
		{
			name:        "handle nil pointer for address",
			input:       &TestStruct{Name: "Charlie", Age: 30, Address: nil},
			makeZeroNil: false,
			ignoreZero:  false,
			expected: map[string]interface{}{
				"name": "Charlie",
				"age":  30,
				"is_employed": false,
				"address":     nil,
			},
			expectError: false,
		},
		{
			name:        "make zero nil",
			input:       &TestStruct{Name: "Dave", Age: 0, IsEmployed: false},
			makeZeroNil: true,
			ignoreZero:  false,
			expected: map[string]interface{}{
                "address": nil,
				"name":        "Dave",
				"age":         nil,
				"is_employed": nil,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := JsonStructToObject(ctx, tt.input, tt.makeZeroNil, tt.ignoreZero)
			if (err != nil) != tt.expectError {
				t.Errorf("JsonStructToObject() error = %v, wantErr %v", err, tt.expectError)
				return
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("JsonStructToObject() got = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestElementsToStringArray(t *testing.T) {
	tests := []struct {
		name     string
		elements []attr.Value
		expected *[]string
	}{
		{
			name:     "valid string elements",
			elements: []attr.Value{types.StringValue("one"), types.StringValue("two"), types.StringValue("three")},
			expected: &[]string{"one", "two", "three"},
		},
		{
			name:     "empty elements",
			elements: []attr.Value{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ElementsToStringArray(tt.elements)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ElementsToStringArray() got = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAttributesToNativeTypes(t *testing.T) {
	tests := []struct {
		name     string
		attrs    map[string]attr.Value
		expected map[string]interface{}
	}{
		{
			name: "all types present",
			attrs: map[string]attr.Value{
				"str":  types.StringValue("test"),
				"int":  types.Int32Value(42),
				"bool": types.BoolValue(true),
			},
			expected: map[string]interface{}{
				"str":  "test",
				"int":  int32(42),
				"bool": true,
			},
		},
		{
			name:     "empty attributes",
			attrs:    map[string]attr.Value{},
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AttributesToNativeTypes(tt.attrs)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("AttributesToNativeTypes() got = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNativeBasicTypedAttributesToTerraform(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name       string
		attrs      map[string]interface{}
		attrTypes  map[string]attr.Type
		expected   map[string]attr.Value
	}{
		{
			name: "string, int32, and bool types",
			attrs: map[string]interface{}{
				"str":  "test",
				"int":  int32(42),
				"bool": true,
			},
			attrTypes: map[string]attr.Type{
				"str":  types.StringType,
				"int":  types.Int32Type,
				"bool": types.BoolType,
			},
			expected: map[string]attr.Value{
				"str":  types.StringValue("test"),
				"int":  types.Int32Value(42),
				"bool": types.BoolValue(true),
			},
		},
		{
			name: "nil values",
			attrs: map[string]interface{}{
				"str":  nil,
				"int":  nil,
				"bool": nil,
			},
			attrTypes: map[string]attr.Type{
				"str":  types.StringType,
				"int":  types.Int32Type,
				"bool": types.BoolType,
			},
			expected: map[string]attr.Value{
				"str":  types.StringNull(),
				"int":  types.Int32Null(),
				"bool": types.BoolNull(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NativeBasicTypedAttributesToTerraform(ctx, tt.attrs, tt.attrTypes)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("NativeBasicTypedAttributesToTerraform() got = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConvertKeysToCamel(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "convert keys to camel case",
			input: map[string]interface{}{
				"first_name": "John",
				"last_name":  "Doe",
			},
			expected: map[string]interface{}{
				"firstName": "John",
				"lastName":  "Doe",
			},
		},
		{
			name: "empty map",
			input: map[string]interface{}{},
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertKeysToCamel(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("convertKeysToCamel() got = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConvertKeysToSnake(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "convert keys to snake case",
			input: map[string]interface{}{
				"firstName": "John",
				"lastName":  "Doe",
			},
			expected: map[string]interface{}{
				"first_name": "John",
				"last_name":  "Doe",
			},
		},
		{
			name: "empty map",
			input: map[string]interface{}{},
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertKeysToSnake(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("convertKeysToSnake() got = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsZero(t *testing.T) {
	tests := []struct {
		name     string
		input    int32
		expected bool
	}{
		{
			name:     "zero value",
			input:    0,
			expected: true,
		},
		{
			name:     "non-zero value",
			input:    5,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsZero(tt.input)
			if result != tt.expected {
				t.Errorf("IsZero() got = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConvertStringList(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name     string
		input    *[]string
		elemType attr.Type
		expected types.List
	}{
		{
			name:     "non-empty list",
			input:    &[]string{"one", "two"},
			elemType: types.StringType,
			expected: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("one"), types.StringValue("two")}),
		},
		{
			name:     "empty list",
			input:    &[]string{},
			elemType: types.StringType,
			expected: types.ListNull(types.StringType),
		},
		{
			name:     "nil list",
			input:    nil,
			elemType: types.StringType,
			expected: types.ListNull(types.StringType),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertStringList(ctx, tt.input, tt.elemType)
			if !result.Equal(tt.expected) {
				t.Errorf("convertStringList() got = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGenericFromObject(t *testing.T) {
	tests := []struct {
		name       string
		mapData    map[string]interface{}
		dto        *TestStruct
		expectError bool
	}{
		{
			name: "valid map to struct",
			mapData: map[string]interface{}{
				"name": "Alice",
				"age":  30,
			},
			dto: &TestStruct{},
			expectError: false,
		},
		{
			name: "invalid json marshaling",
			mapData: map[string]interface{}{
				// This will actually generate an error during marshaling
				"invalid": make(chan int),
			},
			dto: &TestStruct{},
			expectError: true,
		}, 
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GenericFromObject(tt.mapData, tt.dto)
			if (err != nil) != tt.expectError {
				t.Errorf("GenericFromObject() error = %v, wantErr = %v", err, tt.expectError)
			}
		})
	}
}
