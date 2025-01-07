package provider

import (
    "context"
	"reflect"
	"fmt"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"encoding/json"

)

func JsonStructToObject(ctx context.Context, s interface{}, makeZeroNil bool, ignoreZero bool) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	val := reflect.ValueOf(s)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %s", val.Kind())
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {

		field := val.Field(i)
		fieldType := typ.Field(i)

		// Get the json tag
		jsonTag := fieldType.Tag.Get("json")
		if jsonTag == "" {
			continue // Ignore fields without a tag
		}
		// Check for omitempty
		tagParts := strings.Split(jsonTag, ",")
		key := tagParts[0] // The first part is the key

		if field.Kind() == reflect.Ptr {
			if field.IsNil() {
				if ignoreZero {
					continue // Skip nil pointer fields when makeZeroNil is true
				}
				result[key] = nil
				continue
			}
			field = field.Elem()
		}
		fieldValue := field.Interface()

		tflog.Debug(ctx, "KIND OF "+key+" is "+field.Kind().String())
		isEmptyValue := field.IsZero() && field.Kind() != reflect.Int32
		if makeZeroNil && isEmptyValue {
			fieldValue = nil
		}

		isEmptySlice := field.Kind() == reflect.Slice && field.Len() == 0
		if makeZeroNil && isEmptySlice {
			fieldValue = nil
		}

		if ignoreZero && (isEmptyValue || isEmptySlice) {
			continue
		}
		// Handle nested structs
		if field.Kind() == reflect.Struct || (field.Kind() == reflect.Ptr && field.Elem().Kind() == reflect.Struct) {
			nestedValue, err := JsonStructToObject(ctx, field.Interface(), makeZeroNil, ignoreZero)
			if err != nil {
				return nil, err
			}
			fieldValue = nestedValue
		}

		result[key] = fieldValue // Use the actual field value
	}

	return result, nil

}
func ElementsToStringArray(elements []attr.Value) *[]string {
	if len(elements) != 0 {
		elementsArray := []string{}
		for _, v := range elements {
			if val, ok := v.(types.String); ok {
				elementsArray = append(elementsArray, val.ValueString())
			}
		}
		return &elementsArray
	}
	return nil
}

func AttributesToNativeTypes(attrs map[string]attr.Value) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range attrs {
		if val, ok := value.(types.String); ok {
			result[key] = val.ValueString()
		} else if val, ok := value.(types.Int32); ok {
			result[key] = val.ValueInt32()
		} else if val, ok := value.(types.Bool); ok {
			result[key] = val.ValueBool()
		}
	}
	return result

}

func NativeBasicTypedAttributesToTerraform(ctx context.Context, attrs map[string]interface{}, attrTypes map[string]attr.Type) map[string]attr.Value {
	result := make(map[string]attr.Value)

	for targetAttrName, targetAttrType := range attrTypes {
		value, _ := attrs[targetAttrName]
		if targetAttrType == types.StringType {
			if value == nil {
				result[targetAttrName] = types.StringNull()
			} else if val, ok := value.(string); ok {
				result[targetAttrName] = types.StringValue(val)
			} else if val, ok := value.(*string); ok {
				result[targetAttrName] = types.StringPointerValue(val)
			} else {
				tflog.Debug(ctx, "Could not convert "+targetAttrName+" to "+targetAttrType.String())
			}
		} else if targetAttrType == types.Int32Type {
			if value == nil {
				result[targetAttrName] = types.Int32Null()
			} else if val, ok := value.(int32); ok {
				result[targetAttrName] = types.Int32Value(val)
			} else if val, ok := value.(*int32); ok {
				result[targetAttrName] = types.Int32PointerValue(val)
			} else {
				tflog.Debug(ctx, "Could not convert "+targetAttrName+" to "+targetAttrType.String())
			}
		} else if targetAttrType == types.BoolType {
			if value == nil {
				result[targetAttrName] = types.BoolNull()
			} else if val, ok := value.(bool); ok {
				result[targetAttrName] = types.BoolValue(val)
			} else if val, ok := value.(*bool); ok {
				result[targetAttrName] = types.BoolPointerValue(val)
			} else {
				tflog.Debug(ctx, "Could not convert "+targetAttrName+" to "+targetAttrType.String())
			}
		}

	}

	return result

}

func convertKeysToCamel(mapData map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range mapData {
		result[strcase.ToLowerCamel(key)] = value
	}
	return result

}

func convertKeysToSnake(mapData map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range mapData {
		result[strcase.ToSnake(key)] = value
	}
	return result

}

func IsZero[T comparable](v T) bool {
	return v == *new(T)
}

func convertStringList(ctx context.Context, list *[]string, elemType attr.Type) (types.List) {
    var result types.List

	if list != nil && len(*list) > 0 {
		result, _ = types.ListValueFrom(ctx, elemType, list)
	} else {
		result = types.ListNull(elemType)
	}
	return result
}

func GenericFromObject[T any](mapData map[string]interface{}, dto *T) error {
	// Marshal the map to JSON
	data, err := json.Marshal(mapData)
	if err != nil {
		return err
	}

	// Unmarshal the JSON into the provided dto
	if err := json.Unmarshal(data, &dto); err != nil {
		return err
	}

	return nil
}
