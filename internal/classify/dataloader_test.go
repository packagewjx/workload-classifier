package classify

import (
	"fmt"
	"testing"
)

func TestLoadCsv(t *testing.T) {
	loader := &csvLoader{}
	datum, err := loader.Load("../c_15794.csv", []int{0, 1, 5, 6, 7})
	if err != nil {
		t.Error(err)
	}
	fmt.Println(datum)
}
