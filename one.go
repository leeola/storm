package storm

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/boltdb/bolt"
	"github.com/fatih/structs"
)

// GetByIndex returns one record by the specified index
func (s *DB) GetByIndex(index string, value interface{}, to interface{}) error {
	ref := reflect.ValueOf(to)

	if ref.Kind() != reflect.Ptr && structs.IsStruct(to) {
		return errors.New("provided target must be a pointer to struct")
	}

	d := structs.New(to)
	bucketName := d.Name()
	if bucketName == "" {
		return errors.New("provided target must have a name")
	}

	field := d.Field(index)
	tag := field.Tag("storm")
	if tag == "" {
		return fmt.Errorf("index %s doesn't exist", index)
	}

	return s.Bolt.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("bucket %s doesn't exist", bucketName)
		}

		idx := bucket.Bucket([]byte(index))
		if idx == nil {
			return fmt.Errorf("index %s doesn't exist", index)
		}

		val, err := toBytes(value)
		if err != nil {
			return err
		}

		raw := idx.Get(val)
		if raw == nil {
			return errors.New("not found")
		}

		var id []byte

		if tag == "unique" {
			id = raw
		} else if tag == "index" {
			var list [][]byte

			err = json.Unmarshal(raw, &list)
			if err != nil {
				return err
			}

			if list == nil || len(list) == 0 {
				return errors.New("not found")
			}
			id = list[0]
		} else {
			return fmt.Errorf("unsupported struct tag %s", tag)
		}

		raw = bucket.Get(id)
		if raw == nil {
			return errors.New("not found")
		}

		return json.Unmarshal(raw, to)
	})
}