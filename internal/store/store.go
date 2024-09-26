package store

import (
	"encoding/json"
	"fmt"

	"github.com/direktoren/gecholog/internal/gechologobject"
)

type ArrayLog struct {
	Name    string          `json:"name"`
	Details json.RawMessage `json:"details"`
}

func Store(o *gechologobject.GechoLogObject, errObject *gechologobject.GechoLogObject, key string, v any) {
	if key == "" {
		errObject.AssignField("ignore", fmt.Errorf("Empty key in store f").Error())
		return
	}
	// Reused store function
	raw, typeJsonRawMessage := v.(json.RawMessage)
	bytes, typeByteSlice := v.([]byte)
	func() {
		if typeJsonRawMessage {
			err := o.AssignFieldRaw(key, raw)
			if err != nil {
				errObject.AssignField(key, err.Error())
			}
			return
		}
		if typeByteSlice {
			err := o.AssignFieldRaw(key, bytes)
			if err != nil {
				errObject.AssignField(key, err.Error())
			}
			return
		}
		err := o.AssignField(key, v)
		if err != nil {
			errObject.AssignField(key, err.Error())
		}
	}()
}

func StoreInArray(o *gechologobject.GechoLogObject, errObject *gechologobject.GechoLogObject, key string, source *gechologobject.GechoLogObject) {
	// Reused store function

	if source == nil {
		errObject.AssignField(key, fmt.Errorf("Empty source").Error())
		return
	}

	sourceKeys := source.FieldNames()
	if len(sourceKeys) == 0 {
		//errObject.AssignField(key, fmt.Errorf("Empty source").Error())
		return
	}
	newArray := make([]ArrayLog, len(sourceKeys))
	for index, sourceKey := range sourceKeys {
		jrm, err := source.GetField(sourceKey)
		if err != nil {
			errObject.AssignField(key, err.Error())
		}
		newArray[index] = ArrayLog{Name: sourceKey, Details: jrm}
	}
	o.AssignField(key, &newArray)
}
