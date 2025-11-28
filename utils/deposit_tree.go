package utils

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"sync"
)

// DepositContractTreeDepth is the depth of the deposit Merkle tree.
// This is set to 32 as per the Ethereum 2.0 specification, allowing for
// up to 2^32 - 1 deposits.
const DepositContractTreeDepth = 32

// MaxDepositCount is the maximum number of deposits that can be stored.
const MaxDepositCount = (1 << DepositContractTreeDepth) - 1

// ErrMerkleTreeFull is returned when the Merkle tree is at maximum capacity.
var ErrMerkleTreeFull = errors.New("merkle tree full")

// DepositTree implements the incremental Merkle tree algorithm used by
// the Ethereum 2.0 deposit contract. This implementation follows the
// formal verification specification from:
// https://github.com/runtimeverification/verified-smart-contracts/blob/master/deposit/deposit-formal-verification.pdf
//
// The algorithm maintains an O(h) space complexity where h is the tree height,
// and provides O(h) time complexity for both deposits and root computation.
type DepositTree struct {
	branch       [DepositContractTreeDepth][32]byte
	zeroHashes   [DepositContractTreeDepth][32]byte
	depositCount uint64
	mu           sync.RWMutex
}

// NewDepositTree creates and initializes a new DepositTree.
// It computes the zero hashes for the empty sparse Merkle tree.
func NewDepositTree() *DepositTree {
	dt := &DepositTree{}
	dt.initZeroHashes()
	return dt
}

// initZeroHashes computes the hash values for an empty sparse Merkle tree.
// zeroHashes[0] = hash(0, 0) (32 zero bytes)
// zeroHashes[i+1] = hash(zeroHashes[i], zeroHashes[i])
func (dt *DepositTree) initZeroHashes() {
	// zeroHashes[0] is already initialized to all zeros
	for height := 0; height < DepositContractTreeDepth-1; height++ {
		dt.zeroHashes[height+1] = sha256Hash(dt.zeroHashes[height][:], dt.zeroHashes[height][:])
	}
}

// sha256Hash computes sha256(left || right)
func sha256Hash(left, right []byte) [32]byte {
	h := sha256.New()
	h.Write(left)
	h.Write(right)
	var result [32]byte
	copy(result[:], h.Sum(nil))
	return result
}

// Deposit adds a new deposit data root to the Merkle tree.
// This implements the deposit function from the formal specification:
//
//	fun deposit(value: bytes32) -> unit:
//	    assert deposit_count < 2^TREE_HEIGHT - 1
//	    deposit_count += 1
//	    size: int = deposit_count
//	    i: int = 0
//	    while i < TREE_HEIGHT - 1:
//	        if size % 2 == 1:
//	            break
//	        value = hash(branch[i], value)
//	        size /= 2
//	        i += 1
//	    branch[i] = value
func (dt *DepositTree) Deposit(depositDataRoot [32]byte) error {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	if dt.depositCount >= MaxDepositCount {
		return ErrMerkleTreeFull
	}

	dt.depositCount++
	size := dt.depositCount
	node := depositDataRoot

	for height := 0; height < DepositContractTreeDepth; height++ {
		if size&1 == 1 {
			dt.branch[height] = node
			return nil
		}
		node = sha256Hash(dt.branch[height][:], node[:])
		size /= 2
	}

	// This should be unreachable if the algorithm is correct
	return errors.New("deposit: unreachable code")
}

// GetDepositRoot returns the current deposit root hash.
// This implements the get_deposit_root function from the formal specification:
//
//	fun get_deposit_root() -> bytes32:
//	    root: bytes32 = 0
//	    size: int = deposit_count
//	    h: int = 0
//	    while h < TREE_HEIGHT:
//	        if size % 2 == 1: # size is odd
//	            root = hash(branch[h], root)
//	        else:             # size is even
//	            root = hash(root, zerohashes[h])
//	        size /= 2
//	        h += 1
//	    return hash(root, to_little_endian_64(deposit_count), padding)
func (dt *DepositTree) GetDepositRoot() [32]byte {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	var node [32]byte
	size := dt.depositCount

	for height := 0; height < DepositContractTreeDepth; height++ {
		if size&1 == 1 {
			node = sha256Hash(dt.branch[height][:], node[:])
		} else {
			node = sha256Hash(node[:], dt.zeroHashes[height][:])
		}
		size /= 2
	}

	// Mix in the deposit count (as per the Ethereum 2.0 specification)
	return dt.hashTreeRootWithCount(node)
}

// hashTreeRootWithCount mixes the deposit count into the final root hash.
// This matches the Solidity implementation:
// sha256(abi.encodePacked(node, to_little_endian_64(deposit_count), bytes24(0)))
func (dt *DepositTree) hashTreeRootWithCount(node [32]byte) [32]byte {
	h := sha256.New()
	h.Write(node[:])

	// Add deposit count as little-endian 64-bit integer
	var countBytes [8]byte
	binary.LittleEndian.PutUint64(countBytes[:], dt.depositCount)
	h.Write(countBytes[:])

	// Add 24 bytes of padding
	var padding [24]byte
	h.Write(padding[:])

	var result [32]byte
	copy(result[:], h.Sum(nil))
	return result
}

// GetDepositCount returns the current number of deposits.
func (dt *DepositTree) GetDepositCount() uint64 {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	return dt.depositCount
}

// GetDepositCountBytes returns the deposit count encoded as little-endian 64-bit bytes.
func (dt *DepositTree) GetDepositCountBytes() []byte {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	countBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(countBytes, dt.depositCount)
	return countBytes
}

// GetBranch returns a copy of the current branch array.
// This is primarily useful for testing and debugging.
func (dt *DepositTree) GetBranch() [DepositContractTreeDepth][32]byte {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	var result [DepositContractTreeDepth][32]byte
	copy(result[:], dt.branch[:])
	return result
}

// GetZeroHashes returns a copy of the zero hashes array.
// This is primarily useful for testing and debugging.
func (dt *DepositTree) GetZeroHashes() [DepositContractTreeDepth][32]byte {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	var result [DepositContractTreeDepth][32]byte
	copy(result[:], dt.zeroHashes[:])
	return result
}

// DepositDataRoot computes the deposit data root from the deposit data components.
// This follows the SSZ specification for DepositData hash tree root:
//
//	DepositData = {
//	    pubkey: BLSPubkey (48 bytes)
//	    withdrawal_credentials: Bytes32 (32 bytes)
//	    amount: Gwei (uint64 as little-endian 8 bytes)
//	    signature: BLSSignature (96 bytes)
//	}
func DepositDataRoot(pubkey []byte, withdrawalCredentials []byte, amount uint64, signature []byte) ([32]byte, error) {
	if len(pubkey) != 48 {
		return [32]byte{}, errors.New("invalid pubkey length: expected 48 bytes")
	}
	if len(withdrawalCredentials) != 32 {
		return [32]byte{}, errors.New("invalid withdrawal_credentials length: expected 32 bytes")
	}
	if len(signature) != 96 {
		return [32]byte{}, errors.New("invalid signature length: expected 96 bytes")
	}

	// pubkey_root = sha256(pubkey || bytes16(0))
	h := sha256.New()
	h.Write(pubkey)
	var pubkeyPadding [16]byte
	h.Write(pubkeyPadding[:])
	pubkeyRoot := h.Sum(nil)

	// signature_root = sha256(sha256(signature[:64]) || sha256(signature[64:] || bytes32(0)))
	h.Reset()
	h.Write(signature[:64])
	sigLeft := h.Sum(nil)

	h.Reset()
	h.Write(signature[64:])
	var sigPadding [32]byte
	h.Write(sigPadding[:])
	sigRight := h.Sum(nil)

	h.Reset()
	h.Write(sigLeft)
	h.Write(sigRight)
	signatureRoot := h.Sum(nil)

	// amount as little-endian 8 bytes with 24 bytes padding
	var amountBytes [8]byte
	binary.LittleEndian.PutUint64(amountBytes[:], amount)

	// left_branch = sha256(pubkey_root || withdrawal_credentials)
	h.Reset()
	h.Write(pubkeyRoot)
	h.Write(withdrawalCredentials)
	leftBranch := h.Sum(nil)

	// right_branch = sha256(amount || bytes24(0) || signature_root)
	h.Reset()
	h.Write(amountBytes[:])
	var amountPadding [24]byte
	h.Write(amountPadding[:])
	h.Write(signatureRoot)
	rightBranch := h.Sum(nil)

	// deposit_data_root = sha256(left_branch || right_branch)
	h.Reset()
	h.Write(leftBranch)
	h.Write(rightBranch)

	var result [32]byte
	copy(result[:], h.Sum(nil))
	return result, nil
}
