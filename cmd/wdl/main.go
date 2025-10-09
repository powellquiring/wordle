package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"os"
	"runtime/pprof"
	"slices"
	"sort"

	"github.com/powellquiring/wordle/wordle"
	"github.com/urfave/cli/v3" // imports as package "cli"
)

// playWordle with guess/answer pairs provided
func playWordle(globalConfig GlobalConfiguration, answers []string) {
	d := globalConfig.dictionary
	guessAnswers := []wordle.GuessAnswer{}
	for i := 0; i < len(answers); i += 2 {
		guessString := answers[i]
		answerString := answers[i+1]
		if _, ok := d.Word(guessString); !ok {
			panic("guess not in dictionary: " + guessString)
		}
		if _, ok := wordle.StringToAnswer(answerString); !ok {
			panic("answer not in right format r,y,g lik rrggy" + answerString)
		}
		guessAnswers = append(guessAnswers, wordle.GuessAnswer{Guess: guessString, Answer: answerString})
	}
	nextGuess, possibleWords := d.PlayWorldReturnPossible(guessAnswers)
	fmt.Print(d.String(nextGuess), ":")
	for _, word := range d.WordlistStrings(possibleWords) {
		fmt.Print(" ", string(word[:]))
	}
	fmt.Println()
}

type GuessResults struct {
	Guess      string
	Average    float64
	GuessCount []int // number of games for each guess count
}

type Game struct {
	Solution wordle.WordleWord
	Guesses  []wordle.WordleWord
}

const FIRST_DIR = "saved"

func replaceFirstFiles(d *wordle.Dictionary, summary []map[int][]Game) {
	// Create the FIRST_DIR directory if it doesn't exist
	if err := os.MkdirAll(FIRST_DIR, 0755); err != nil {
		panic("failed to create directory " + FIRST_DIR + ": " + err.Error())
	}

	for _, sortedGames := range summary {
		// Get the first word from the first game with 1 guess (if it exists)
		var firstWord string
		if games, ok := sortedGames[1]; ok && len(games) > 0 {
			firstWord = d.String(games[0].Guesses[0])
		} else {
			panic("no first word")
		}

		filename := FIRST_DIR + "/" + firstWord + ".json"
		fmt.Println("writing", filename)

		// Create the JSON structure: map[string][][]string
		jsonData := make(map[string][][]string)

		// Create array of arrays - each inner array contains solutions with same number of guesses
		var allSequences [][]string
		for numberOfGuesses, games := range sortedGames {
			if numberOfGuesses <= 1 {
				// do not include the first guess that matches the solution
				continue
			}
			for _, game := range games {
				guesses := make([]string, 0)
				for guessCount, guess := range game.Guesses {
					if guessCount == 0 {
						continue // first guess is the firstWord do not need to repeat it for every set of guesses
					}
					guesses = append(guesses, d.String(guess))
				}
				allSequences = append(allSequences, guesses)
			}
		}

		jsonData[firstWord] = allSequences

		// Write JSON to file
		jsonBytes, err := json.MarshalIndent(jsonData, "", "  ")
		if err != nil {
			panic("failed to marshal JSON for " + firstWord + ": " + err.Error())
		}

		if err := os.WriteFile(filename, jsonBytes, 0644); err != nil {
			panic("failed to write file " + filename + ": " + err.Error())
		}
	}
}
func outputFinalSummary(d *wordle.Dictionary, summary []map[int][]Game) {
	const MAX_GUESS_COUNT = 7
	games := make([]GuessResults, 0)
	for _, sortedGames := range summary {
		totalGuesses := 0
		totalGames := 0
		guessCount := make([]int, MAX_GUESS_COUNT)
		guess := ""
		for i := range MAX_GUESS_COUNT - 1 { // 1, 7
			numberGuessesAtCount := i + 1 // 1, 7
			gameGuesses, ok := sortedGames[numberGuessesAtCount]
			if ok {
				guess = d.String(gameGuesses[0].Guesses[0])
			} else {
				gameGuesses = make([]Game, 0)
			}
			totalGuesses += numberGuessesAtCount * len(gameGuesses)
			totalGames += len(gameGuesses)
			guessCount[numberGuessesAtCount] = len(gameGuesses)
		}
		games = append(games, GuessResults{guess, float64(totalGuesses) / float64(totalGames), guessCount})
	}

	fmt.Println("By average")
	sort.Slice(games, func(i, j int) bool {
		return games[i].Average < games[j].Average
	})
	printGames(games)

	fmt.Println("By guess 6")
	sort.Slice(games, func(i, j int) bool {
		return games[i].GuessCount[6] < games[j].GuessCount[6]
	})
	printGames(games)

	fmt.Println("By guess 5+6")
	sort.Slice(games, func(i, j int) bool {
		return games[i].GuessCount[6]+games[i].GuessCount[5] < games[j].GuessCount[6]+games[j].GuessCount[5]
	})
	printGames(games)

	fmt.Println("By guess 2+3")
	sort.Slice(games, func(i, j int) bool {
		return games[i].GuessCount[2]+games[i].GuessCount[3] > games[j].GuessCount[2]+games[j].GuessCount[3]
	})
	printGames(games)
	fmt.Println("done")
}

func printGames(gameResults []GuessResults) {
	for _, gameResult := range gameResults {
		fmt.Printf("%s %f ", gameResult.Guess, gameResult.Average)
		for guessCount, numberGuessesAtCount := range gameResult.GuessCount {
			if guessCount < 1 {
				continue
			}
			fmt.Printf("%d ", numberGuessesAtCount)
		}
		fmt.Println()
	}
}

func simulate(globalConfig GlobalConfiguration, oneGame bool, replaceFirst bool, firstWordsStrings []string, solutionStrings []string) {
	d := globalConfig.dictionary
	solutions := d.WordlistEmpty()
	if len(solutionStrings) == 0 {
		solutions = d.WordlistAll()
	} else {
		for _, solutionString := range solutionStrings {
			solution, ok := d.Word(solutionString)
			if !ok {
				panic("solution not in dictionary: " + solutionString)
			}
			solutions.Insert(solution)
		}
	}

	/*
		var bar *progressbar.ProgressBar
		if globalConfig.progress {
			bar = progressbar.Default(int64(solutions.Len()))
		} else {
			bar = progressbar.DefaultSilent(int64(solutions.Len()))
		}
	*/

	// firstWords is a slice (outer loop) of slices (guesses for one game), see SimulateOneGameGivenFirstWord)
	// if oneGame is true then there is only one outer loop and one or more guesses for the game.  If oneGame is false
	// then there are multiple outer loops (games) and only one guess for each game.
	var initialGuesesList [][]wordle.WordleWord
	if len(firstWordsStrings) == 0 {
		if oneGame {
			panic("must supply first words for one game")
		}
		wordList := d.WordlistAll()
		for _, word := range wordList.Range {
			initialGuesesList = append(initialGuesesList, []wordle.WordleWord{word})
		}
	} else {
		if oneGame {
			initialGuesesList = append(initialGuesesList, d.StringsToWordSlice(firstWordsStrings))
		} else {
			for _, firstWordString := range firstWordsStrings {
				firstWord, ok := d.Word(firstWordString)
				if !ok {
					panic("first word not in dictionary: " + firstWordString)
				}
				initialGuesesList = append(initialGuesesList, []wordle.WordleWord{firstWord})
			}
		}
	}

	summary := make([]map[int][]Game, len(initialGuesesList))
	for outerLoopCount, initialGuesses := range initialGuesesList {
		sortedGames := make(map[int][]Game)
		fmt.Println("outer loop count, limit:", outerLoopCount, len(initialGuesesList), d.WordSliceToStrings(initialGuesses))
		for solutionCount, solution := range solutions.Range {
			guesses := wordle.SimulateOneGameGivenFirstWord(d, solution, initialGuesses)
			fmt.Print(solutionCount, solutions.Len(), " ", d.String(solution), ":")
			for _, guess := range guesses {
				fmt.Print(" ", d.String(guess))
			}
			fmt.Println()

			sortedGames[len(guesses)] = append(sortedGames[len(guesses)], Game{solution, guesses})
		}
		fmt.Println(d.String(initialGuesses[0]), "---------------------")

		// create slice of number of guesses
		keys := slices.Collect(maps.Keys(sortedGames))

		// Sort the slice of keys
		sort.Ints(keys)

		for _, numGuesses := range keys {
			games := sortedGames[numGuesses]
			fmt.Println(numGuesses, len(games), " ---------------------")
			for _, game := range games {
				fmt.Print(d.String(game.Solution), ":")
				for _, guess := range game.Guesses {
					fmt.Print(" ", d.String(guess))
				}
				fmt.Println()
			}
		}
		summary[outerLoopCount] = sortedGames
	}
	outputFinalSummary(d, summary)
	if replaceFirst {
		replaceFirstFiles(d, summary)
	}
}

func first(globalConfig GlobalConfiguration) {
	d := globalConfig.dictionary
	wordScoreSorter := d.SortedGuesses(d.WordlistAll(), 0)
	// for sortedGuessScores.Len() > 0 {
	for _, item := range *wordScoreSorter {
		fmt.Println(d.String(item.Value), item.Score)
	}
}

func cpuProfile() func() {
	f, err := os.Create("cpu.prof")
	if err != nil {
		panic(err)
	}
	pprof.StartCPUProfile(f)
	return pprof.StopCPUProfile
}

type GlobalConfiguration struct {
	dictionary *wordle.Dictionary
	progress   bool
}

func globalCofiguration(count int, progress bool) GlobalConfiguration {
	if count == 0 {
		count = len(wordle.SortedWordleDictionary())
	}
	dictionary := wordle.NewDictionary(wordle.SortedWordleDictionary()[0:count])
	return GlobalConfiguration{
		dictionary: dictionary,
		progress:   progress,
	}
}

func main() {
	count := 0
	progress := false
	profile := false
	firstWord := ""
	// command specific flags
	simulateOneGame := false
	simulateReplace := false
	cmd := &cli.Command{
		Name:  "wdl",
		Usage: "wordle",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:        "count",
				Value:       0,
				Aliases:     []string{"c"},
				Usage:       "number of words, 0 is all words",
				Destination: &count,
			},
			&cli.BoolFlag{
				Name:        "progress",
				Value:       false,
				Aliases:     []string{"p"},
				Usage:       "show progress bar",
				Destination: &progress,
			},
			&cli.BoolFlag{
				Name:        "profile",
				Value:       false,
				Usage:       "store profile data to analyze",
				Destination: &profile,
			},
			&cli.StringFlag{
				Name:        "first",
				Value:       "",
				Aliases:     []string{"f"},
				Usage:       "first word to guess, default is 'raise', only used with sim command",
				Destination: &firstWord,
			},
		},
		Commands: []*cli.Command{
			{
				Name: "play",
				Usage: `play a game of wordle against the by entering pairs of [guess answer]...
				https://www.nytimes.com/games/wordle/index.html
				`,
				Action: func(ctx context.Context, cmd *cli.Command) error {

					if profile {
						def := cpuProfile()
						defer def()
					}

					if cmd.NArg()%2 != 0 {
						return cli.Exit("must have pairs of guess answer", 1)
					} else if cmd.NArg() < 2 {
						return cli.Exit("must have at least one guess answer", 2)
					} else {
						playWordle(globalCofiguration(count, progress), cmd.Args().Slice())
					}
					return nil
				},
			},
			{
				Name: "sim",
				Usage: `sim -a [firstword][solution] ...
				Simulate multiple games by specifying a list of solutions for each game.  Firstword will be the first
				string supplied unless the -a flag is set. If no solutions are provided,
				simulate solutions for all words.  All words can be cut back by using the -count global flag for testing.
				`,
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Usage:   "--first first1 first2 first3 ...",
						Name:    "first",
						Aliases: []string{"f"},
					},
					&cli.BoolFlag{
						Name:  "one",
						Value: false,
						Usage: `wdl sim -one --first abyss --first create --first canal cacao
						play one game of wordle.  Use each of the first words in order for the initial guesses in the game.
						Useful for finding performance problems for combinartions of guesses.`,
						Destination: &simulateOneGame,
					},
					&cli.BoolFlag{
						Name:  "replace",
						Value: false,
						Usage: `wdl sim -replace [--first abyss]
						incompatible with the -one flag. Simulate game(s) of wordle and replace the contents in the saved/ directory.
						has a file for each first word, first.json, containg the optimial play for the word.  First.json files will be used
						to improve performance of the play command.`,
						Destination: &simulateReplace,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					firstWords := cmd.StringSlice("first")
					solutions := cmd.Args().Slice()
					if profile {
						def := cpuProfile()
						defer def()
					}
					simulate(globalCofiguration(count, progress), simulateOneGame, simulateReplace, firstWords, solutions)
					return nil
				},
			},
			{
				Name: "first",
				Usage: `first
				Sort first words by simple score
				`,
				Action: func(ctx context.Context, cmd *cli.Command) error {
					if profile {
						def := cpuProfile()
						defer def()
					}
					first(globalCofiguration(count, progress))
					return nil
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
