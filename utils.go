package main

import (
	"errors"
	"fmt"
	"reflect"
)

func Copy(toValue interface{}, fromValue interface{}) (err error) {
	// Recover from any panics and return them as an error
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprint("panic in Copy: ", r))
		}
	}()

	toType := reflect.TypeOf(toValue)
	fromType := reflect.TypeOf(fromValue)

	// Check if toValue is a pointer
	if toType.Kind() != reflect.Ptr {
		return errors.New("toValue must be a pointer")
	}

	// Get the value that toValue points to
	toElem := reflect.ValueOf(toValue).Elem()

	// Get the value of fromValue
	fromElem := reflect.ValueOf(fromValue)

	// If fromValue is a pointer, get the value it points to
	if fromType.Kind() == reflect.Ptr {
		fromElem = fromElem.Elem()
	}

	// If toElem is nil (in case of pointer types), create a new value of the appropriate type
	if toElem.Kind() == reflect.Ptr && toElem.IsNil() {
		toElem.Set(reflect.New(toElem.Type().Elem()))
	}

	// If toElem is a pointer, we need to set the value it points to
	if toElem.Kind() == reflect.Ptr {
		if toElem.Elem().Type().AssignableTo(fromElem.Type()) {
			toElem.Elem().Set(fromElem)
		} else {
			return errors.New("fromValue type is not assignable to toValue's element type")
		}
	} else {
		// Check if types are assignable for non-pointer types
		if !fromElem.Type().AssignableTo(toElem.Type()) {
			return errors.New("fromValue type is not assignable to toValue type")
		}
		toElem.Set(fromElem)
	}

	return nil
}
