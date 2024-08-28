package main

import "github.com/jinzhu/copier"

func Clone[T any](obj *T) *T {
	var newObj *T = new(T)
	copier.Copy(newObj, obj)
	return newObj
}
