#!/usr/bin/env bash

cd $(dirname $0)

for MODEL in cogito mistral-nemo phi4-fncall qwen2.5-coder gemma3-tools polaris; do
  ./run-$MODEL-numbers.sh
  ollama stop $(ollama ps | tail -n1 | cut -f1 -d" ")
  sleep 5
done
