// Copyright 2013 Rainer Volz.
// See the LICENSE file for license information.

package core

/*
#include <stdlib.h>
int	qosLevels[] = {0, 1, 1000, 34000, 65000};
*/
import "C"
import "testing"
import "unsafe"

func testCIntArray2Go(t *testing.T) {
	arraySize := 5
	arrayContent := [5]int{0, 1, 1000, 34000, 65000}
	goIntArray := cIntArray2Go(unsafe.Pointer(&C.qosLevels), 5)
	if len(goIntArray) != arraySize {
		t.Error("Bad array size", arraySize, len(goIntArray))
	}
	for i := 0; i < arraySize; i++ {
		if goIntArray[i] != arrayContent[i] {
			t.Error("Bad array content", arrayContent[i], goIntArray[i])
		}
	}
}
