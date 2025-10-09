package wordle

import (
	"container/heap"
	"container/list"
	"fmt"
	"sort"

	//	"github.com/bits-and-blooms/bitset"
	"github.com/powellquiring/wordle/bitset"
	"github.com/powellquiring/wordle/gowordle"
)

// MinHeap is a generic min-heap that can store any type T.
type MinHeap[T any] struct {
	data []T
	less func(a, b T) bool
}

func (h *MinHeap[T]) Len() int           { return len(h.data) }
func (h *MinHeap[T]) Less(i, j int) bool { return h.less(h.data[i], h.data[j]) }
func (h *MinHeap[T]) Swap(i, j int)      { h.data[i], h.data[j] = h.data[j], h.data[i] }

// Push adds an element to the heap.
func (h *MinHeap[T]) Push(x any) {
	h.data = append(h.data, x.(T))
}

// Pop removes the highest-priority element.
func (h *MinHeap[T]) Pop() any {
	n := len(h.data)
	item := h.data[n-1]
	h.data = h.data[0 : n-1]
	return item
}

// An Item is something we manage in a priority queue.
type Item struct {
	Value WordleWord // The value of the item; arbitrary.
	Score int        // The priority of the item in the queue.
	// The index is needed by update and is maintained by the heap.Interface methods.
}

func NewMinHeapWordleWordPriority() *MinHeap[Item] {
	ret := &MinHeap[Item]{
		data: []Item{},
		less: func(a, b Item) bool {
			// The priority queue will be based on the 'priority' field.
			return a.Score < b.Score
		},
	}
	heap.Init(ret)
	return ret
}

type WordScore struct {
	Value WordleWord
	Score uint16 // The priority of the item lower is better.
}

type WordScoreSorter []WordScore

/*
	func NewWordScoreHeap(len int) *WordScoreSorter {
		wordScores := make([]WordScore, 0, len)
		return (*WordScoreSorter)(&wordScores)
	}
*/
func (h *WordScoreSorter) Reset() {
	*h = (*h)[:0]
}

func (h *WordScoreSorter) Push(x WordScore) {
	if len(*h) >= cap(*h) {
		panic("WordScoreHeap is full")
	}
	*h = append(*h, x)
}

var fullAnswerCacheHitCount int
var fullAnswerCacheMissCount int
var cacheSolutionGuess bool = true

func (d *Dictionary) GoWordleSliceToWordList(goMatching []gowordle.WordleWord) *WordList {
	possibleWords := d.WordlistEmpty()
	for _, goWord := range goMatching {
		wordString := string(goWord[:])
		word, ok := d.Word(wordString)
		if !ok {
			panic("word not in dictionary: " + wordString)
		}
		possibleWords.Insert(word)
	}
	return possibleWords
}

func (d *Dictionary) GetFullAnswerForDictionary(wordlist *WordList, solution WordleWord, guess WordleWord) FullAnswer {
	goAnswer := gowordle.WordleAnswer2(gowordle.WordleWord([]rune(d.String(solution))), gowordle.WordleWord([]rune(d.String(guess))))
	goColor := string(goAnswer.Colors[:])
	goMatching := d.matcher.Matching2(goAnswer)
	color, ok := StringToAnswer(goColor)
	if !ok {
		panic("Color not valid: " + goColor)
	}
	possibleWords := d.GoWordleSliceToWordList(goMatching)
	return FullAnswer{AnswerColor: color, AnswerMatching: possibleWords}
}

const fullAnswerCachePrintCount int = 0

// given a wordlist a solution and a guess return the answer and new wordlist
func (d *Dictionary) GetFullAnswerForDictionaryWithCache(wordlist *WordList, solution WordleWord, guess WordleWord) FullAnswer {
	if (fullAnswerCachePrintCount > 0) && (fullAnswerCacheHitCount+fullAnswerCacheMissCount)%10_000_000 == 0 {
		fmt.Println("fullAnswerCacheHitCount:", fullAnswerCacheHitCount, "fullAnswerCacheMissCount:", fullAnswerCacheMissCount)
	}
	if d.fullAnswerCache[solution][guess].AnswerMatching != nil {
		fullAnswerCacheHitCount++
		return d.fullAnswerCache[solution][guess]
	} else {
		fullAnswerCacheMissCount++
		fullAnswer := d.GetFullAnswerForDictionary(wordlist, solution, guess)
		d.fullAnswerCache[solution][guess] = fullAnswer
		return fullAnswer
	}
}

func (d *Dictionary) GetFullAnswer(wordlist *WordList, solution WordleWord, guess WordleWord, possibleWords *WordList) FullAnswer {
	var fullAnswer FullAnswer
	if cacheSolutionGuess {
		fullAnswer = d.GetFullAnswerForDictionaryWithCache(wordlist, solution, guess)
	} else {
		fullAnswer = d.GetFullAnswerForDictionary(wordlist, solution, guess)
	}
	bs := (*bitset.BitSet)(fullAnswer.AnswerMatching)
	bs.IntersectionInPlace((*bitset.BitSet)(wordlist), (*bitset.BitSet)(possibleWords))
	return FullAnswer{AnswerColor: fullAnswer.AnswerColor, AnswerMatching: (*WordList)(possibleWords)}
}

func (d *Dictionary) GetFullAnswerLength(wordlist *WordList, solution WordleWord, guess WordleWord) int {
	var fullAnswer FullAnswer
	if cacheSolutionGuess {
		fullAnswer = d.GetFullAnswerForDictionaryWithCache(wordlist, solution, guess)
	} else {
		fullAnswer = d.GetFullAnswerForDictionary(wordlist, solution, guess)
	}
	bs := (*bitset.BitSet)(fullAnswer.AnswerMatching)
	return int(bs.IntersectionBitCount((*bitset.BitSet)(wordlist)))
}

var subscoreCacheHit int
var subscoreCacheMiss int
var subscoreCache = make(map[WordList](int))

func subscoreCacheGet(matching *WordList) (int, bool) {
	if false && ((subscoreCacheHit+subscoreCacheMiss)%10_000_000 == 0) {
		fmt.Println("Subscore cache hit/miss: ", subscoreCacheHit, subscoreCacheMiss)
	}
	ret, ok := subscoreCache[*matching]
	if ok {
		subscoreCacheHit++
	} else {
		subscoreCacheMiss++
	}
	return ret, ok
}

func subscoreCacheSet(matching *WordList, subscore int) {
	subscoreCache[*matching] = subscore
}

var wordScoreSorter WordScoreSorter

// LRUCacheNode holds the key and the WordleWord value to traverse the list of words in MRU order.
// over half of the time it is possible to exit early when exaiming all guesses
type LRUCacheNode struct {
	key   int
	value WordleWord
}

// StandardLRUCache is an unlocked (non-thread-safe) LRU cache storing WordleWord.
// Convention: Head (Front) = MRU, Tail (Back) = LRU.
type StandardLRUCache struct {
	capacity int
	cache    map[int]*list.Element
	ll       *list.List
}

// NewLRUCache creates a new cache and populates it with WordleWord values 0 to capacity-1.
// Initial order: Head->[999, 998, ..., 0]<-Tail.
func NewLRUCache(capacity int) *StandardLRUCache {
	c := &StandardLRUCache{
		capacity: capacity,
		cache:    make(map[int]*list.Element, capacity),
		ll:       list.New(),
	}

	// Populate the cache. The value stored is a WordleWord (uint16).
	for i := 0; i < capacity; i++ {
		// Cast the integer index to the custom WordleWord type
		wordValue := WordleWord(i)

		newNode := &LRUCacheNode{key: i, value: wordValue}
		newElem := c.ll.PushFront(newNode) // Latest added item goes to the front (MRU)
		c.cache[i] = newElem
	}
	// MRU is WordleWord(999), LRU is WordleWord(0).
	return c
}

// Touch moves the element to the Head (MRU).
func (c *StandardLRUCache) Touch(key int) {
	if elem, hit := c.cache[key]; hit {
		// Move the element to the Head (Front) to mark it as MRU.
		c.ll.MoveToFront(elem)
	} else {
		panic("key not found")
	}
}

// RangeMRU creates a snapshot slice and returns an iterator over that slice.
// This ensures the iteration order is fixed when RangeMRU is called.
func (c *StandardLRUCache) RangeMRU() func(yield func(value WordleWord) bool) {
	// 1. Read all values into a new slice (the immutable snapshot).
	snapshot := make([]WordleWord, 0, c.ll.Len())
	for elem := c.ll.Front(); elem != nil; elem = elem.Next() {
		snapshot = append(snapshot, elem.Value.(*LRUCacheNode).value)
	}

	// 2. Return the iterator function which traverses the immutable slice.
	return func(yield func(value WordleWord) bool) {
		for _, value := range snapshot {
			if !yield(value) {
				return
			}
		}
	}
}

var lruCache *StandardLRUCache

// Return a list sourted by score.  The score is the number of possible words that remain after the guess when the guess
// is used in all possible words.  Lower scores are better.
// In some cases it is possible to find the perfect guess.  In this case a list of one word is returned and the score is
// the score that would be returned by NextGuessSearch.
func (d *Dictionary) SortedGuesses(possibleWords *WordList, depth int) *WordScoreSorter {
	// possible words are allocated here to minimize the number of initializations
	wordScoreSorter := &wordScoreSorter
	if len(*wordScoreSorter) == 0 {
		*wordScoreSorter = make([]WordScore, d.Len())
	}
	wordScoreSorter.Reset()
	if lruCache == nil {
		lruCache = NewLRUCache(d.Len())
	}
	lenPossibleWords := possibleWords.Len()
	usedGuesses := make([]bool, d.Len())

	// first try all the possible words most of the time a perfect guess is found in the possible words
	for _, guess := range possibleWords.Range {
		score := 0
		for _, solution := range possibleWords.Range {
			matchingLen := d.GetFullAnswerLength(possibleWords, solution, guess)
			score += matchingLen
		}
		if score == lenPossibleWords {
			// this is the best possible guess there is no need to try any more.
			// score is 100 for the correct guess and 200 for the rest.
			return &WordScoreSorter{{Value: guess, Score: uint16((100 + 200*(lenPossibleWords-1)) / lenPossibleWords)}}
		}
		wordScoreSorter.Push(WordScore{Value: guess, Score: uint16(score - 2)})
		usedGuesses[guess] = true
	}
	for guess := range lruCache.RangeMRU() {
		if usedGuesses[guess] {
			continue // the possible words have already been evaluated
		}
		score := 0
		for _, solution := range possibleWords.Range {
			matchingLen := d.GetFullAnswerLength(possibleWords, solution, guess)
			score += matchingLen
		}
		if score <= lenPossibleWords {
			// this guess is perfect, return only this guess.  The score is 100 to narrow it down to 1 more
			// so 2000 total
			// It is more likley to be a good guess next time, so move it to the front of the LRU cache
			lruCache.Touch(int(guess))
			return &WordScoreSorter{{Value: guess, Score: uint16(200)}}
		}
		wordScoreSorter.Push(WordScore{Value: guess, Score: uint16(score)})
	}
	sort.Slice(*wordScoreSorter, func(i, j int) bool {
		return (*wordScoreSorter)[i].Score < (*wordScoreSorter)[j].Score
	})
	// bummer need to return a long list of guesses, score will be correctly calculated by NextGuessSearch
	return wordScoreSorter
}

var depthExceededCount int

// starting with a subset of the dictionary words (wordlist) give a score to each word in the dictionary
// based on "guess score" for that word.
func (d *Dictionary) NextGuessSearch(possibleWords *WordList, depth int) (int, WordleWord) {
	const INIFINITY_SCORE = 1000000

	// possible words are allocated here to minimize the number of initializations
	var fullanswerPossibleWords WordList

	if depth > 14 {
		depthExceededCount++
		fmt.Println("depth:", depth, "possibleWords:", depthExceededCount, possibleWords.Len(), d.WordlistStrings(possibleWords))
	}
	possibleWordsLen := possibleWords.Len()
	if possibleWordsLen == 0 {
		panic("possibleWords is empty")
	}
	if possibleWordsLen == 1 {
		return 100, possibleWords.FirstWord() // just guess it
	}
	if possibleWordsLen == 2 {
		// if there are two words choose either of the words and the guesses will be 1 if the right guess and 2 if the wrong guess
		return 150, possibleWords.FirstWord()
	}

	// Using the possible words the best guess is the matching one for one solution and 2 for the rest of the solutions.
	// if retScore, retWordsWithScore, ok := scoreForPossibleWords(game.id); ok {
	// 	return retScore, retWordsWithScore
	// }
	possibleWordsSet := make(map[WordleWord]bool)
	for _, guess := range possibleWords.Range {
		possibleWordsSet[guess] = true
	}
	guessesInPossibleWords := make([]WordleWord, 0)
	guessesNotInPossibleWords := make([]WordleWord, 0)
	flagGuessCountCutOff := 110
	maxGuessCount := 150
	if maxGuessCount > d.Len() {
		maxGuessCount = d.Len()
	}
	wordScoreSorter := d.SortedGuesses(possibleWords, depth)
	if len(*wordScoreSorter) == 1 {
		//SortedGuesses found the perfect guess.
		return (int)((*wordScoreSorter)[0].Score), (*wordScoreSorter)[0].Value
	}
	// sortedScores := d.SortedGuesses(possibleWords, wordScoreSorter)
	// for guessCount := 0; (len(*wordScoreSorter) > 0) && (guessCount < maxGuessCount); guessCount++ {
	for itemCount, item := range *wordScoreSorter {
		if itemCount >= maxGuessCount {
			break
		}
		guess := item.Value
		if _, ok := possibleWordsSet[guess]; ok {
			guessesInPossibleWords = append(guessesInPossibleWords, guess)
		} else {
			guessesNotInPossibleWords = append(guessesNotInPossibleWords, guess)
		}
	}

	bestScore := INIFINITY_SCORE
	// assume best possible score is a correct guess (100) and getting all the rest of the solutions in 2 guesses
	var bestGuess WordleWord
	for guessCount, guess := range append(guessesInPossibleWords, guessesNotInPossibleWords...) {
		//guessesLen := len(guessesInPossibleWords)
		score := 0 // running average
		guessInPossibleWordsRemaining := false
		bestPossibleScore := (100 + 200*(possibleWordsLen-1)) / possibleWordsLen
		if guessCount < len(guessesInPossibleWords) {
			// this is a guess that is in the possible words
			guessInPossibleWordsRemaining = true
		} else {
			bestPossibleScore = 200
		}
		if bestScore <= bestPossibleScore {
			// not going to add any more identical scores to the best guess list
			break
		}
		for count, solution := range possibleWords.Range {
			fullAnswer := d.GetFullAnswer(possibleWords, solution, guess, &fullanswerPossibleWords)
			matching := fullAnswer.AnswerMatching
			matchingLen := matching.Len()
			//	if len(matching) == possibleWordsLen {
			//		// not narrowing it down any this solution so it is a bad guess, go to next guess
			//		score = INIFINITY_SCORE
			//		break
			//	}
			if depth > 17 || matchingLen == possibleWordsLen {
				score = INIFINITY_SCORE
				break // this guess is bad move to the next guess
			}
			// Score the guess for this solution
			guessSolutionScore := 100 // one guess is 100 points
			if (matchingLen == 1) && (matching.FirstWord() == guess) {
				guessInPossibleWordsRemaining = false // this is the correct guess
			} else {
				var subscore int
				if subscoreCached, ok := subscoreCacheGet(matching); ok {
					subscore = subscoreCached
				} else {
					subscore, _ = d.NextGuessSearch(matching, depth+1)
					if subscore != INIFINITY_SCORE {
						subscoreCacheSet(matching, subscore)
					} else {
						fmt.Println("subscor cache miss")
					}
				}
				guessSolutionScore += subscore
			}
			score = score + ((guessSolutionScore - score) / (count + 1)) // running average

			// 200 is the best for the remaining words, if the current average plus best possible result for the remaining words
			// is alread over that may as well quit
			bestPossibleScoreForThisGuess := ((score * (count + 1)) + (200 * (possibleWordsLen - (count + 1)))) / possibleWordsLen
			if guessInPossibleWordsRemaining {
				// if a correct guess is coming up then the score is 100 for the matching guess and 200 for the rest
				if possibleWordsLen < (count + 2) {
					panic("bad count")
				}
				bestPossibleScoreForThisGuess = ((score * (count + 1)) + 100 + (200 * (possibleWordsLen - (count + 2)))) / possibleWordsLen
			}
			if bestPossibleScoreForThisGuess > bestScore {
				score = bestPossibleScoreForThisGuess // greater then bestScore is all that matters
				break
			}
		}
		if score == INIFINITY_SCORE {
			//if (bestGuess == nil) && (guessCount == (guessesLen - 1)) {
			//	fmt.Println("no score. depth:", depth, "guessCount:", guessCount, "possibleWords:", possibleWordsLen, d.WordlistStrings(possibleWords))
			//}
			continue // this guess is bad move to the next guess
		}

		if score < bestScore {
			if guessCount > flagGuessCountCutOff {
				fmt.Println("old/new:", bestScore, score, "depth:", depth, "guessCount:", guessCount, "guess:", d.String(guess), "possibleWords:", possibleWordsLen, d.WordlistStrings(possibleWords))
			}
			bestScore = score
			bestGuess = guess

			//} else if score == bestScore {
			//	bestGuess.Insert(guess)
		}
		if false && (depth == 0) {
			fmt.Println("depth guessCount score", depth, guessCount, score)
		}
	}
	return bestScore, bestGuess
}

// simulate one game given the first word and the solution
func SimulateOneGameGivenFirstWord(dictionary *Dictionary, solution WordleWord, initialGuesses []WordleWord) []WordleWord {
	// possible words are allocated here to minimize the number of initializations
	var fullanswerPossibleWords WordList
	guesses := []WordleWord{}
	matchingWords := dictionary.WordlistAll()
	for guessCount := range 8 {
		var nextGuess WordleWord
		if guessCount < len(initialGuesses) {
			nextGuess = initialGuesses[guessCount]
		} else {
			nextGuess = dictionary.NextGuess(matchingWords)
		}
		guesses = append(guesses, nextGuess)
		fullAnswer := dictionary.GetFullAnswer(matchingWords, solution, nextGuess, &fullanswerPossibleWords)
		matchingWords = fullAnswer.AnswerMatching
		if matchingWords.Len() == 1 {
			words := matchingWords.Words()
			if words[0] == solution {
				if words[0] != nextGuess {
					// if the solution is the nextGuess it was already added, do not need to add it again
					guesses = append(guesses, words[0])
				}
				return guesses
			} else {
				panic("unexpected Simulate end")
			}
		}
	}
	panic("unexpected Simulate end")
}

type GuessAnswer struct {
	Guess  string
	Answer string
}

// play wordle against the computer providing the current board state
// return the next best answer
func (d *Dictionary) PlayWorldReturnPossible(guessAnswers []GuessAnswer) (WordleWord, *WordList) {
	goMatching := []gowordle.WordleWord{}

	var game *gowordle.WordleMatcher
	for guessCount, guessAnswer := range guessAnswers {
		if guessCount == 0 {
			game = d.matcher
		} else {
			game = gowordle.NewWordleMatcher(goMatching)
		}
		goGuess := gowordle.WordleWord([]rune(guessAnswer.Guess))
		goAnswer := gowordle.WordleWord([]rune(guessAnswer.Answer))
		goMatching = game.Matching(goGuess, goAnswer)
	}
	possibleAnswers := d.GoWordleSliceToWordList(goMatching)
	ret := d.NextGuess(possibleAnswers)
	return ret, possibleAnswers
}
