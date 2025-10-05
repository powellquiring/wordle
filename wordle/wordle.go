package wordle

import (
	"strconv"

	"github.com/powellquiring/wordle/bitset"

	//	"github.com/bits-and-blooms/bitset"
	"github.com/powellquiring/wordle/gowordle"
)

// WordleWord is an index into the dictionary
type WordleWord uint16

// Answer is a bitset where there are 2 bits for each Color
type Answer uint16

type Color uint16
type WordList bitset.BitSet

const (
	Red Color = iota
	Yellow
	Green
)

type Dictionary struct {
	words           []string
	stringToWord    map[string]WordleWord
	matcher         *gowordle.WordleMatcher
	fullAnswerCache [][]FullAnswer
}

type FullAnswer struct {
	AnswerColor    Answer
	AnswerMatching *WordList
}

func StringToAnswer(colors string) (Answer, bool) {
	ret := Answer(0)
	ok := true
	for _, color := range colors {
		ret <<= 2
		switch color {
		case 'r':
			ret |= Answer(Red)
		case 'y':
			ret |= Answer(Yellow)
		case 'g':
			ret |= Answer(Green)
		default:
			ok = false
		}
	}
	return ret, ok
}

func (a Answer) String() string {
	answer := Color(a)
	ret := ""
	for range 5 {
		color := answer & 3
		answer >>= 2
		switch color {
		case Red:
			ret = "r" + ret
		case Yellow:
			ret = "y" + ret
		case Green:
			ret = "g" + ret
		default:
			panic("Can not parse Color: " + strconv.Itoa(int(color)))
		}
	}
	if answer != 0 {
		panic("Can not parse Answer extra bits: " + strconv.Itoa(int(answer)))
	}
	return ret
}

func NewDictionary(strings []string) *Dictionary {
	ret := &Dictionary{words: strings}
	ret.stringToWord = make(map[string]WordleWord)
	for i, word := range strings {
		ret.stringToWord[word] = WordleWord(i)
	}
	ret.matcher = gowordle.NewWordleMatcher(gowordle.StringsToWordleWords(strings))
	stringsLen := len(strings)
	ret.fullAnswerCache = make([][]FullAnswer, stringsLen)
	for i := range ret.fullAnswerCache {
		ret.fullAnswerCache[i] = make([]FullAnswer, stringsLen)
	}
	return ret
}

func (d *Dictionary) Len() int {
	return len(d.words)
}

func (d *Dictionary) WordlistAll() *WordList {
	wordsLen := uint(len(d.words))
	ret := bitset.New(wordsLen)
	ret.SetAll(wordsLen)
	return (*WordList)(ret)
}

func (d *Dictionary) WordlistFromStrings(strings []string) *WordList {
	ret := d.WordlistEmpty()
	for _, word := range strings {
		wordleWord, ok := d.Word(word)
		if !ok {
			panic("word not in dictionary: " + word)
		}
		ret.Insert(wordleWord)
	}
	return ret
}

func (d *Dictionary) WordlistEmpty() *WordList {
	ret := bitset.New(uint(len(d.words)))
	return (*WordList)(ret)
}

func (d *Dictionary) Word(wordleWordString string) (WordleWord, bool) {
	ret, ok := d.stringToWord[wordleWordString]
	return ret, ok
}

func (d *Dictionary) String(WordleWord WordleWord) string {
	return d.words[WordleWord]
}

func (d *Dictionary) WordlistStrings(wordlist *WordList) []string {
	ret := []string{}
	for _, word := range wordlist.Range {
		ret = append(ret, d.String(word))
	}
	return ret
}
func (d *Dictionary) StringsToWordSlice(strings []string) []WordleWord {
	ret := []WordleWord{}
	for _, word := range strings {
		wordleWord, ok := d.Word(word)
		if !ok {
			panic("word not in dictionary: " + word)
		}
		ret = append(ret, wordleWord)
	}
	return ret
}
func (d *Dictionary) WordSliceToStrings(wordSlice []WordleWord) []string {
	ret := []string{}
	for _, word := range wordSlice {
		ret = append(ret, d.String(word))
	}
	return ret
}

// given a wordlist a solution and a guess return the answer and new wordlist
func (d *Dictionary) NextGuess(wordlist *WordList) WordleWord {
	_, guesses := d.NextGuessSearch(wordlist, 0)
	return guesses.Words()[0]
}

func (wl *WordList) Range(yield func(i int, wordleWord WordleWord) bool) {
	bs := (*bitset.BitSet)(wl)
	i := 0
	for wordleWord, ok := bs.NextSet(0); ok; wordleWord, ok = bs.NextSet(wordleWord + 1) {
		// Call the yield function (which is the loop body)
		if !yield(i, WordleWord(wordleWord)) {
			return // Stop iteration if yield returns false (e.g., break in the loop)
		}
		i++
	}
}

func (wl *WordList) Words() []WordleWord {
	ret := []WordleWord{}
	for _, wordleWord := range wl.Range {
		ret = append(ret, wordleWord)
	}
	return ret
}
func (wl *WordList) FirstWord() WordleWord {
	bs := (*bitset.BitSet)(wl)
	wordleWord, ok := bs.NextSet(0)
	if !ok {
		panic("no first word")
	}
	return WordleWord(wordleWord)
}

func (wl *WordList) Len() int {
	bs := (*bitset.BitSet)(wl)
	return int(bs.Count())
}

func (wordlist *WordList) Insert(word WordleWord) {
	bs := (*bitset.BitSet)(wordlist)
	bs.Set(uint(word))
}
