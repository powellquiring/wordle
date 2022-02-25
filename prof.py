#!/usr/bin/env python
import cProfile
import pstats
from pstats import SortKey
import w

ofile = "/tmp/stats"
cProfile.run('w.perf2()', ofile)

p = pstats.Stats(ofile)
p.sort_stats(SortKey.CUMULATIVE).print_stats(10)
