#!/bin/sh
nohup ./chainwatch --repo=/data/sdb/lotus-user-1/.lotus --db="postgres://postgres:pgSS0000@10.1.30.2/lotus_testnet3_dev?sslmode=disable" run >>chainwatch.log 2>&1 &

# ./chainwatch --db="postgres://postgres:pgSS0000@10.1.30.2/lotus_chain?sslmode=disable" dot 85000 95000 > blocks.dot && dot blocks.dot -Tsvg -Grankdir=TB > blocks.svg
