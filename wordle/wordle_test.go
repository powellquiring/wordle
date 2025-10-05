package wordle

import (
	"fmt"
	"testing"
)

func stringToWordOrPanic(d *Dictionary, s string) WordleWord {
	word, ok := d.Word(s)
	if !ok {
		panic("word not in dictionary: " + s)
	}
	return word
}
func BenchmarkSimulate(t *testing.B) {
	d := NewDictionary(SortedWordleDictionary()[:])
	guesses := SimulateOneGameGivenFirstWord(d, stringToWordOrPanic(d, "hello"), []WordleWord{stringToWordOrPanic(d, "raise")})
	fmt.Println(guesses)
}
