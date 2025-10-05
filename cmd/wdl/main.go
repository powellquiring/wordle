package main

import (
	"container/heap"
	"context"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
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

func simulate(globalConfig GlobalConfiguration, oneGame bool, firstWordsStrings []string, solutionStrings []string) {
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

	type Game struct {
		Solution wordle.WordleWord
		Guesses  []wordle.WordleWord
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
	// if oneGame is true then there is only one outer loop

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

			if _, ok := sortedGames[len(guesses)]; !ok {
				sortedGames[len(guesses)] = make([]Game, 0)
			}
			sortedGames[len(guesses)] = append(sortedGames[len(guesses)], Game{solution, guesses})
		}
		fmt.Println("---------------------")

		// create slice of number of guesses
		keys := make([]int, 0, len(sortedGames))
		for k := range sortedGames {
			keys = append(keys, k)
		}
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
	}
}

func first(globalConfig GlobalConfiguration) {
	d := globalConfig.dictionary
	sortedGuessScores := d.SortedGuesses(d.WordlistAll())
	for sortedGuessScores.Len() > 0 {
		item := heap.Pop(sortedGuessScores).(wordle.Item)
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
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					firstWords := cmd.StringSlice("first")
					solutions := cmd.Args().Slice()
					if profile {
						def := cpuProfile()
						defer def()
					}
					simulate(globalCofiguration(count, progress), simulateOneGame, firstWords, solutions)
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
