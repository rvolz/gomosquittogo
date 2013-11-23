// Copyright 2013 Rainer Volz.
// See the LICENSE file for license information.

package core

import "unsafe"

// Transform a C pointer of an integer array to its Go equivalent
// see http://stackoverflow.com/questions/14826319/go-cgo-how-do-you-use-a-c-array-passed-as-a-pointer
func cIntArray2Go(cIntArr unsafe.Pointer, cIntArrSize int) []int {
	var goIntArray = make([]int, cIntArrSize)
	var p *int32 = (*int32)(cIntArr)
	for i := 0; i < cIntArrSize; i++ {
		goIntArray[i] = (int)(*p)
		p = (*int32)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + unsafe.Sizeof(*p)))
	}
	return goIntArray
}

// Check if an integer array contains an integer value
func containsInt(key int, haystack []int) bool {
	for i := 0; i < len(haystack); i++ {
		if haystack[i] == key {
			return true
		}
	}
	return false
}
