#!/usr/bin/env bash

# This simnet harness sets up 4 Firo nodes and a set of harness controls
# Each node has a prepared, encrypted, empty wallet

# Wallets updated to new protocol for spark 2024-01-08

SYMBOL="firo"
DAEMON="firod"
CLI="firo-cli"
RPC_USER="user"
RPC_PASS="pass"
ALPHA_LISTEN_PORT="53764"
ALPHA_RPC_PORT="53768"
WALLET_PASSWORD="abc"
ALPHA_MINING_ADDR="TDEWAEjLGfBcoqnn68nWE1UJgD8hqEbfAY"

# Background watch mining in window 5 by default:  
# 'export NOMINER="1"' or uncomment this line to disable
#NOMINER="1"

set -evx

NODES_ROOT="${HOME}/dextest/${SYMBOL}"
rm -rf "${NODES_ROOT}"
# The firo directory tree is now clean
SOURCE_DIR=$(pwd)

ALPHA_DIR="${NODES_ROOT}/alpha"
mkdir -p ${ALPHA_DIR}
HARNESS_DIR="${NODES_ROOT}/harness-ctl"
mkdir -p ${HARNESS_DIR}

ALPHA_CLI_CFG="-rpcport=${ALPHA_RPC_PORT} -regtest=1 -rpcuser=user -rpcpassword=pass"

# DONE can be used in a send-keys call along with a `wait-for firo` command to
# wait for process termination.
DONE="; tmux wait-for -S ${SYMBOL}"
WAIT="wait-for ${SYMBOL}"

SESSION="${SYMBOL}-harness" # `firo-harness`

SHELL=$(which bash)

################################################################################
# Load prepared wallet.
################################################################################
echo "Loading prepared, encrypted but empty wallet"

mkdir -p "${ALPHA_DIR}/regtest"
cp "${SOURCE_DIR}/alpha_wallet.dat" "${ALPHA_DIR}/regtest/wallet.dat"

################################################################################
# Write config files.
################################################################################
# echo "Writing node config files"

# cat > "${ALPHA_DIR}/alpha.conf" <<EOF
# rpcuser=user
# rpcpassword=pass
# datadir=${ALPHA_DIR}
# txindex=1
# port=${ALPHA_LISTEN_PORT}
# regtest=1
# rpcport=${ALPHA_RPC_PORT}
# dandelion=0
# EOF

################################################################################
# Start Tmux.
################################################################################
cd ${HARNESS_DIR} && tmux new-session -d -s $SESSION $SHELL

################################################################################
# Setup the harness-ctl directory in window target 0
################################################################################
cd ${HARNESS_DIR}

tmux rename-window -t $SESSION:0 'harness-ctl'
tmux send-keys -t $SESSION:0 "set +o history" C-m
tmux send-keys -t $SESSION:0 "cd ${HARNESS_DIR}" C-m
sleep 1

cat > "./alpha" <<EOF
#!/usr/bin/env bash
${CLI} ${ALPHA_CLI_CFG} "\$@"
EOF
chmod +x "./alpha"

cat > "./mine-alpha" <<EOF
#!/usr/bin/env bash
${CLI} ${ALPHA_CLI_CFG} generatetoaddress \$1 ${ALPHA_MINING_ADDR}
EOF
chmod +x "./mine-alpha"

cat > "${HARNESS_DIR}/quit" <<EOF
#!/usr/bin/env bash
tmux send-keys -t $SESSION:0 C-c
tmux send-keys -t $SESSION:1 C-c
if [ -z "$NOMINER" ] ; then
  tmux send-keys -t $SESSION:2 C-c
fi  
tmux wait-for alpha${SYMBOL}
# seppuku
tmux kill-session
EOF
chmod +x "${HARNESS_DIR}/quit"

################################################################################
# Start the alpha node in window target 1.
################################################################################
tmux new-window -t $SESSION:1 -n 'alpha' $SHELL
tmux send-keys -t $SESSION:1 "set +o history" C-m
tmux send-keys -t $SESSION:1 "cd ${ALPHA_DIR}" C-m

echo "Starting simnet alpha node"
tmux send-keys -t $SESSION:1 "${DAEMON} -rpcuser=user -rpcpassword=pass \
  -rpcport=${ALPHA_RPC_PORT} -datadir=${ALPHA_DIR} -txindex=1 -regtest=1 \
  -debug=rpc -debug=net -debug=mempool -debug=walletdb -debug=addrman -debug=mempoolrej \
  -whitelist=127.0.0.0/8 -whitelist=::1 \
  -port=${ALPHA_LISTEN_PORT} -fallbackfee=0.00001 -dandelion=0 \
  -printtoconsole; \
  tmux wait-for -S alpha${SYMBOL}" C-m
sleep 3

################################################################################
# Generate blocks in window target 0
################################################################################
echo "Unlocking mining wallet"
tmux send-keys -t $SESSION:0 "./alpha walletpassphrase ${WALLET_PASSWORD} 100000000 ${DONE}" C-m

echo "Generating 333 blocks for alpha"
tmux send-keys -t $SESSION:0 "./alpha generatetoaddress 333 ${ALPHA_MINING_ADDR} > /dev/null ${DONE}" C-m

################################################################################
# Setup watch background miner in window target 2 -- if required
################################################################################
if [ -z "$NOMINER" ] ; then
  tmux new-window -t $SESSION:2 -n "miner" $SHELL
  tmux send-keys -t $SESSION:2 "cd ${HARNESS_DIR}" C-m
  tmux send-keys -t $SESSION:2 "watch -n 60 ./mine-alpha 1" C-m
fi

######################################################################################
# Reenable history select the harness control window & attach to the control target #
######################################################################################
tmux send-keys -t $SESSION:0 "set -o history" C-m
tmux select-window -t $SESSION:0
tmux attach-session -t $SESSION
