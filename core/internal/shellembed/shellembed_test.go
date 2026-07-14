package shellembed

import "testing"

func TestAvailableFalseWithoutTag(t *testing.T) {
	if Available() {
		t.Fatal("Available() should be false in untagged test builds")
	}
}
