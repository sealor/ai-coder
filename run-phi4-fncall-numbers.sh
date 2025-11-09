#!/usr/bin/env bash

cd $(dirname $0)

PROMPT='Read the listed numbers in "numbers.txt" and correct the file if numbers are missing'

cat >numbers.txt <<EOF
Number list 1:
1
2
4
5
6
7
8
9

Number list 2:
1
2
3
4
6
7
8
9
EOF

. ../Ollama-connection.sh
set -x
go run main.go -model phi4-fncall -tools -message "$PROMPT" -session-file phi4-fncall-numbers.yaml
