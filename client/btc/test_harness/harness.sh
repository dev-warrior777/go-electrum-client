#!/usr/bin/env bash

# Tmux script that sets up a simnet harness for BTC regtest simnet.

SYMBOL="btc"
DAEMON="bitcoind"
CLI="bitcoin-cli"
RPC_USER="user"
RPC_PASS="pass"
ALPHA_LISTEN_PORT="20575"
ALPHA_RPC_PORT="20556"
WALLET_PASSWORD="abc"
ALPHA_WALLET_SEED="cMndqchcXSCUQDDZQSKU2cUHbPb5UfFL9afspxsBELeE6qx6ac9n"
ALPHA_MINING_ADDR="bcrt1qy7agjj62epx0ydnqskgwlcfwu52xjtpj36hr0d"
EXTRA_ARGS="--blockfilterindex --peerblockfilters --rpcbind=0.0.0.0 --rpcallowip=0.0.0.0/0"
CREATE_DEFAULT_WALLET="1"

# Run the harness
HARNESS_VER="1" # for outdated chain archive detection







set -ex
NODES_ROOT=~/dextest/${SYMBOL}
rm -rf "${NODES_ROOT}"

ALPHA_DIR="${NODES_ROOT}/alpha"
BETA_DIR="${NODES_ROOT}/beta"
HARNESS_DIR="${NODES_ROOT}/harness-ctl"

echo "Writing node config files"
mkdir -p "${HARNESS_DIR}"
mkdir -p "${ALPHA_DIR}"

WALLET_PASSWORD="abc"

ALPHA_CLI_CFG="-rpcwallet= -rpcport=${ALPHA_RPC_PORT} -regtest=1 -rpcuser=user -rpcpassword=pass"

# DONE can be used in a send-keys call along with a `wait-for btc` command to
# wait for process termination.
DONE="; tmux wait-for -S ${SYMBOL}"
WAIT="wait-for ${SYMBOL}"

SESSION="${SYMBOL}-harness"

export SHELL=$(which bash)


cd ${NODES_ROOT} && tmux new-session -d -s $SESSION $SHELL

################################################################################
# Write config files.
################################################################################

# These config files aren't actually used here, but can be used by other
# programs. I would use them here, but bitcoind seems to have some issues
# reading from the file when using regtest.

cat > "${ALPHA_DIR}/alpha.conf" <<EOF
rpcuser=user
rpcpassword=pass
datadir=${ALPHA_DIR}
txindex=1
port=${ALPHA_LISTEN_PORT}
regtest=1
rpcport=${ALPHA_RPC_PORT}
EOF

################################################################################
# Start the alpha node.
################################################################################

tmux rename-window -t $SESSION:0 'alpha'
tmux send-keys -t $SESSION:0 "set +o history" C-m
tmux send-keys -t $SESSION:0 "cd ${ALPHA_DIR}" C-m
echo "Starting simnet alpha node"
tmux send-keys -t $SESSION:0 "${DAEMON} -rpcuser=user -rpcpassword=pass \
  -rpcport=${ALPHA_RPC_PORT} -datadir=${ALPHA_DIR} \
  -debug=rpc -debug=net -debug=mempool -debug=walletdb -debug=addrman -debug=mempoolrej \
  -whitelist=127.0.0.0/8 -whitelist=::1 \
  -txindex=1 -regtest=1 -port=${ALPHA_LISTEN_PORT} -fallbackfee=0.00001 \
  ${EXTRA_ARGS}; tmux wait-for -S alpha${SYMBOL}" C-m
sleep 3

################################################################################
# Setup the harness-ctl directory
################################################################################

tmux new-window -t $SESSION:2 -n 'harness-ctl' $SHELL
tmux send-keys -t $SESSION:2 "set +o history" C-m
tmux send-keys -t $SESSION:2 "cd ${HARNESS_DIR}" C-m
sleep 1

cd ${HARNESS_DIR}

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
tmux wait-for alpha${SYMBOL}
# seppuku
tmux kill-session
EOF
chmod +x "${HARNESS_DIR}/quit"

  ################################################################################
  # Have to generate a block before calling sethdseed
  ################################################################################
  echo "Generating the genesis block"
  tmux send-keys -t $SESSION:2 "./alpha generatetoaddress 1 ${ALPHA_MINING_ADDR}${DONE}" C-m\; ${WAIT}
  sleep 2

  #################################################################################
  # Alpha node's wallet
  ################################################################################

  # Create this as a "blank" wallet since sethdseed will follow.
  tmux send-keys -t $SESSION:2 "./alpha -named createwallet wallet_name= blank=true passphrase=\"${WALLET_PASSWORD}\" load_on_startup=true${DONE}" C-m\; ${WAIT}

  tmux send-keys -t $SESSION:2 "./alpha walletpassphrase ${WALLET_PASSWORD} 100000000${DONE}" C-m\; ${WAIT}

  echo "Setting private keys for alpha"
  tmux send-keys -t $SESSION:2 "./alpha sethdseed true ${ALPHA_WALLET_SEED}${DONE}" C-m\; ${WAIT}

  echo "Generating 200 blocks for alpha"
  tmux send-keys -t $SESSION:2 "./alpha generatetoaddress 200 ${ALPHA_MINING_ADDR} > /dev/null${DONE}" C-m\; ${WAIT}

  #################################################################################
  # make smaller utxos
  ################################################################################

  for i in 10 18 5 7 1 15 3 25
  do
    tmux send-keys -t $SESSION:2 "./alpha sendtoaddress ${ALPHA_MINING_ADDR} ${i}${DONE}" C-m\; ${WAIT}
  done


set +x

# Reenable history
tmux send-keys -t $SESSION:2 "set -o history" C-m

# Miner
if [ -z "$NOMINER" ] ; then
  tmux new-window -t $SESSION:3 -n "miner" $SHELL
  tmux send-keys -t $SESSION:3 "cd ${HARNESS_DIR}" C-m
  tmux send-keys -t $SESSION:3 "watch -n 15 ./mine-alpha 1" C-m
fi

tmux select-window -t $SESSION:2
tmux attach-session -t $SESSION
