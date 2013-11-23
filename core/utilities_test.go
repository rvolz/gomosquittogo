// Copyright 2013 Rainer Volz.
// See the LICENSE file for license information.

package core

import "testing"

/*
Tests for the utilities of Mosquitto MQTT Go wrapper.
Since they use cgo we have to put them in non-_test.go files.
See http://golang.org/misc/cgo/test/cgo_test.go
*/

func TestCIntArray2Go(t *testing.T) { testCIntArray2Go(t) }

func TestContainsInt(t *testing.T) {
	haystack := []int{1, 2, 3, 4, 5}
	if !containsInt(3, haystack) {
		t.Error("key not found in integer array")
	}
}
