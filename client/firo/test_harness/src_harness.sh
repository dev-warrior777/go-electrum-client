#!/bin/bash

# run as: source ./src_harness.sh

# all testing is against my bitcoin binary install .. your PATH may be different
export PATH=$PATH:$HOME/firo/bin
which firod
$(pwd)/harness.sh


