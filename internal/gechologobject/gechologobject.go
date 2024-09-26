package gechologobject

import (
	"encoding/json"
	"fmt"
)

type GechoLogObject struct {
	fields map[string]json.RawMessage
}

func New() GechoLogObject {
	o := GechoLogObject{fields: map[string]json.RawMessage{}}
	return o
}
func (o *GechoLogObject) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.fields)
}

func (o *GechoLogObject) UnmarshalJSON(data []byte) error {
	if o.fields == nil {
		o.fields = make(map[string]json.RawMessage)
	}
	return json.Unmarshal(data, &o.fields)
}

// Can be used to create valid emty json string
func ValueForEmptyJson() json.RawMessage {
	return json.RawMessage("{}")
}

func (o *GechoLogObject) IsEmpty() bool {
	return len(o.fields) == 0
}

// Prints a convenient string
func (o *GechoLogObject) DebugString() string {
	str := ""
	for field, value := range o.fields {
		str += fmt.Sprintf("%s: %s ", field, string(value))
	}

	return str
}

// assign key-value par for field->jrm
func (o *GechoLogObject) AssignFieldRaw(field string, jrm json.RawMessage) error {
	if !json.Valid(jrm) {
		return fmt.Errorf("Not a valid json")
	}
	o.fields[field] = jrm
	return nil
}

// Assign key = field to value = json.Marshalled v
func (o *GechoLogObject) AssignField(field string, v any) error {
	bytes, err := json.Marshal(v)
	if err != nil {
		return err
	}
	o.AssignFieldRaw(field, bytes)
	return nil
}

// Getter for field
func (o *GechoLogObject) GetField(field string) (json.RawMessage, error) {
	if o.fields == nil {
		return nil, fmt.Errorf("gechologobject->GetField: Uninitialized fields")
	}
	jsonRawMessage, exists := o.fields[field]
	if !exists {
		return nil, fmt.Errorf("gechologobject->GetField: key %s does not exist", field)
	}
	return jsonRawMessage, nil
}

// returns copy of a with f as filter/transformer 1-to-1
func transformedCopy(g GechoLogObject, f func(header string, jrm json.RawMessage) json.RawMessage) GechoLogObject {
	copyG := New()
	for h, jrm := range g.fields {
		newJRM := f(h, jrm)
		if len(newJRM) != 0 {
			copyG.fields[h] = newJRM
		}
	}
	return copyG
}

// Returns a slice with the list of field names
func (g *GechoLogObject) FieldNames() []string {
	fieldNames := []string{}
	for name, _ := range g.fields {
		fieldNames = append(fieldNames, name)
	}
	return fieldNames
}

// Creates a new object from g without fields in the slice of strings
func Filter(g GechoLogObject, fieldsToInclude []string) GechoLogObject {
	includeMap := map[string]struct{}{}
	for _, fieldName := range fieldsToInclude {
		includeMap[fieldName] = struct{}{}
	}
	return transformedCopy(g, func(h string, values json.RawMessage) json.RawMessage {
		_, exists := includeMap[h]
		if exists {
			return values
		}
		return nil
	})
}

// Creates a new g with replacing overlappning fields from o
func Replace(g GechoLogObject, o GechoLogObject) GechoLogObject {
	return transformedCopy(g, func(h string, jrm json.RawMessage) json.RawMessage {
		newJRM, exists := o.fields[h]
		if exists {
			return newJRM
		}
		return jrm
	})
}

// Creats a new g + Adds non-overlapping fields from o
func AppendNew(g GechoLogObject, o GechoLogObject) GechoLogObject {
	gAppend := transformedCopy(g, func(h string, jrm json.RawMessage) json.RawMessage { return jrm }) // Clean copy
	for h, v := range o.fields {
		_, alreadyexists := gAppend.fields[h]
		if !alreadyexists {
			gAppend.fields[h] = v
		}
	}
	return gAppend
}
