# wordl
Simulate the game of wordle to find best starting words and best guesses
# estimate memory
Cach indexed by solution and guess.
- 2309 dictionary words
- 2309/64 = 37 uint64
- 37 uint == 296 bytes (37 uint64)
 bytes per entry (wordlist) plus 2 bytes for color or 5 * 64bit words
- 2500 * 2500 * 300 = 187,500,000 bytes = 180 MB

