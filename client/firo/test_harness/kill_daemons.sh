#!/usr/bin/env bash
# bunuh djinn
# quick & dirty daemon killer
#
# TEST ONLY
tmux kill-session -t firo-harness
killall -9 firod
sleep 1
