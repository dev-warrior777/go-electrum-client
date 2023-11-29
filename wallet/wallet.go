package wallet

import (
	"errors"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	hd "github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

type WalletConfig struct {
	// The blockchain, Bitcoin, Dash, etc
	Chain CoinType

	// Network parameters. Set mainnet, testnet using this.
	Params *chaincfg.Params

	// Store the seed in encrypted storage
	StoreEncSeed bool

	// Location of the data directory
	DataDir string

	// An implementation of the Datastore interface
	DB Datastore

	// The default fee-per-byte for each level
	LowFee    uint64
	MediumFee uint64
	HighFee   uint64

	// The highest allowable fee-per-byte
	MaxFee uint64

	// If not testing do not overwrite existing wallet files
	Testing bool
}

type ElectrumWallet interface {

	// Start the wallet
	Start()

	// Return the creation date of the wallet
	CreationDate() time.Time

	// Return the network parameters
	Params() *chaincfg.Params

	// Returns the type of crytocurrency this wallet implements
	CurrencyCode() string

	// Check if this amount is considered dust < 1000 sats/equivalent for now
	IsDust(amount int64) bool

	// GetUnusedAddress returns an address suitable for receiving payments.
	// `purpose` specifies whether the address should be internal or external.
	// This function will return the same address so long as that address is
	// not invloved in a transaction. Whenever the returned address has it's
	// first payment tx this function should start returning a new, unused
	// address.
	GetUnusedAddress(purpose KeyPurpose) (btcutil.Address, error)

	// Marks the address as used (involved in at least one transaction)
	MarkAddressUsed(address btcutil.Address) error

	// CreateNewAddress returns a new, never-before-returned address.
	// CAUTION: This will be outside the gap limit.     [deprecated]
	CreateNewAddress(purpose KeyPurpose) btcutil.Address

	// DecodeAddress parses the address string and return an address.
	DecodeAddress(addr string) (btcutil.Address, error)

	// ScriptToAddress takes a raw output script (the full script, not just a
	// hash160) and returns the corresponding address.
	ScriptToAddress(script []byte) (btcutil.Address, error)

	// Turn the given address into an output script
	AddressToScript(address btcutil.Address) ([]byte, error)

	// Add a subscribe script to the wallet. These addresses will be used to
	// subscribe to ElectrumX and get notifications back from ElectrumX
	// when coins are received. If already stored this is a no-op.
	AddSubscribeScript(script []byte) error

	// Returns all the watched scripts in db.
	ListSubscribeScripts() ([][]byte, error)

	// Returns if the wallet has the HD key for the given address
	HasAddress(address btcutil.Address) bool

	// Returns a list of addresses for this wallet
	ListAddresses() []btcutil.Address

	// Balance returns the confirmed balance for the wallet.
	// For utxo based wallets, if a spend of confirmed coins is made, the resulting "change"
	// should be also counted as confirmed even if the spending transaction is unconfirmed.
	//
	// This command uses the local wallet. We can also get from ElectrumX.
	Balance() (int64, int64)

	// Returns a list of transactions for this wallet
	Transactions() ([]Txn, error)

	// Does the wallet have a specific transaction?
	HasTransaction(txid chainhash.Hash) bool

	// Get info on a specific transaction
	GetTransaction(txid chainhash.Hash) (Txn, error)

	// Return the confirmed txids and heights for an address
	GetAddressHistory(address btcutil.Address) ([]AddressHistory, error)

	// Add a transaction to the database
	AddTransaction(tx *wire.MsgTx, height int64, timestamp time.Time) error

	// Make a new spending transaction
	Spend(amount int64, toAddress btcutil.Address, feeLevel FeeLevel, referenceID string, spendAll bool) (*chainhash.Hash, error)

	// Calculates the estimated size of the transaction and returns the total fee for the given feePerByte
	EstimateFee(ins []TransactionInput, outs []TransactionOutput, feePerByte uint64) uint64

	// Build a transaction that sweeps all coins from an address. If it is a p2sh multisig, the redeemScript must be included
	SweepAddress(ins []TransactionInput, address btcutil.Address, key *hd.ExtendedKey, redeemScript []byte, feeLevel FeeLevel) (*chainhash.Hash, error)

	// Generate a multisig script from public keys. If a timeout is included the returned script should be a timelocked
	// escrow which releases using the timeoutKey.
	// GenerateMultisigScript should deterministically create a redeem script and address from the information provided.
	// This method should be strictly limited to taking the input data, combining it to produce the redeem script and
	// address
	GenerateMultisigScript(keys []hd.ExtendedKey, threshold int, timeout time.Duration, timeoutKey *hd.ExtendedKey) (addr btcutil.Address, redeemScript []byte, err error)

	// Create a signature for a multisig transaction
	CreateMultisigSignature(ins []TransactionInput, outs []TransactionOutput, key *hd.ExtendedKey, redeemScript []byte, feePerByte uint64) ([]Signature, error)

	// Combine signatures and optionally broadcast
	Multisign(ins []TransactionInput, outs []TransactionOutput, sigs1 []Signature, sigs2 []Signature, redeemScript []byte, feePerByte uint64, broadcast bool) ([]byte, error)

	// Update the height of the tip of the headers chain
	UpdateTip(newTip int64)

	// Cleanly disconnect from the wallet
	Close()
}

// Errors
var (
	ErrDustAmount error = errors.New("amount is below network dust treshold")
	// ErrInsufficientFunds is returned when the wallet is unable to send the
	// amount specified due to the balance being too low
	ErrInsufficientFunds = errors.New("ERROR_INSUFFICIENT_FUNDS")

	// ErrWalletFnNotImplemented is returned from some unimplemented functions.
	// This is due to a concrete wallet not implementing the finctionality or
	// temporarily during development.
	ErrWalletFnNotImplemented = errors.New("wallet function is not implemented")
)

type FeeLevel int

const (
	PRIOIRTY       FeeLevel = 0
	NORMAL         FeeLevel = 1
	ECONOMIC       FeeLevel = 2
	FEE_BUMP       FeeLevel = 3
	SUPER_ECONOMIC FeeLevel = 4
)

// The end leaves on the HD wallet have only two possible values. External keys are those given
// to other people for the purpose of receiving transactions. These may include keys used for
// refund addresses. Internal keys are used only by the wallet, primarily for change addresses
// but could also be used for shuffling around UTXOs.
type KeyPurpose int

const (
	EXTERNAL KeyPurpose = 0
	INTERNAL KeyPurpose = 1
	// Aliases
	RECEIVING = EXTERNAL
	CHANGE    = INTERNAL
)

type AddressHistory struct {
	Height int64
	TxHash chainhash.Hash
}

type TransactionOutput struct {
	Address btcutil.Address
	Value   int64
	Index   uint32
	OrderID string
}

type TransactionInput struct {
	OutpointHash  []byte
	OutpointIndex uint32
	LinkedAddress btcutil.Address
	Value         int64
	OrderID       string
}

// This object contains a single signature for a multisig transaction. InputIndex specifies
// the index for which this signature applies.
type Signature struct {
	InputIndex uint32
	Signature  []byte
}
