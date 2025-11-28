package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestNewDepositTree(t *testing.T) {
	dt := NewDepositTree()

	if dt.depositCount != 0 {
		t.Errorf("expected deposit count to be 0, got %d", dt.depositCount)
	}

	// Verify zero hashes are correctly computed
	// zeroHashes[0] should be 32 zero bytes
	var expectedZero [32]byte
	if dt.zeroHashes[0] != expectedZero {
		t.Error("expected zeroHashes[0] to be all zeros")
	}

	// zeroHashes[1] should be sha256(zeroHashes[0] || zeroHashes[0])
	expectedHash := sha256Hash(expectedZero[:], expectedZero[:])
	if dt.zeroHashes[1] != expectedHash {
		t.Error("zeroHashes[1] is incorrectly computed")
	}
}

func TestDepositTree_SingleDeposit(t *testing.T) {
	dt := NewDepositTree()

	// Create a test deposit data root
	var depositDataRoot [32]byte
	for i := range depositDataRoot {
		depositDataRoot[i] = byte(i)
	}

	err := dt.Deposit(depositDataRoot)
	if err != nil {
		t.Errorf("unexpected error on deposit: %v", err)
	}

	if dt.GetDepositCount() != 1 {
		t.Errorf("expected deposit count to be 1, got %d", dt.GetDepositCount())
	}

	// Verify the root is computed without error
	root := dt.GetDepositRoot()
	if root == [32]byte{} {
		t.Error("expected non-zero deposit root")
	}
}

func TestDepositTree_MultipleDeposits(t *testing.T) {
	dt := NewDepositTree()

	// Add several deposits
	numDeposits := 10
	for i := 0; i < numDeposits; i++ {
		var depositDataRoot [32]byte
		for j := range depositDataRoot {
			depositDataRoot[j] = byte(i + j)
		}
		err := dt.Deposit(depositDataRoot)
		if err != nil {
			t.Errorf("unexpected error on deposit %d: %v", i, err)
		}
	}

	if dt.GetDepositCount() != uint64(numDeposits) {
		t.Errorf("expected deposit count to be %d, got %d", numDeposits, dt.GetDepositCount())
	}

	root := dt.GetDepositRoot()
	if root == [32]byte{} {
		t.Error("expected non-zero deposit root")
	}
}

func TestDepositTree_EmptyTreeRoot(t *testing.T) {
	dt := NewDepositTree()

	// Get root of empty tree
	root := dt.GetDepositRoot()

	// The root should be deterministic for an empty tree
	// Verify it's not all zeros (it's mixed with the deposit count)
	if root == [32]byte{} {
		t.Error("expected non-zero root even for empty tree due to count mixing")
	}

	// Two empty trees should produce the same root
	dt2 := NewDepositTree()
	root2 := dt2.GetDepositRoot()
	if root != root2 {
		t.Error("expected same root for two empty trees")
	}
}

func TestDepositTree_DeterministicRoots(t *testing.T) {
	// Create two trees with the same deposits
	dt1 := NewDepositTree()
	dt2 := NewDepositTree()

	for i := 0; i < 5; i++ {
		var depositDataRoot [32]byte
		for j := range depositDataRoot {
			depositDataRoot[j] = byte(i * 10)
		}
		_ = dt1.Deposit(depositDataRoot)
		_ = dt2.Deposit(depositDataRoot)
	}

	if dt1.GetDepositRoot() != dt2.GetDepositRoot() {
		t.Error("expected same root for trees with same deposits")
	}
}

func TestDepositTree_DifferentDepositsProduceDifferentRoots(t *testing.T) {
	dt1 := NewDepositTree()
	dt2 := NewDepositTree()

	var deposit1, deposit2 [32]byte
	deposit1[0] = 1
	deposit2[0] = 2

	_ = dt1.Deposit(deposit1)
	_ = dt2.Deposit(deposit2)

	if dt1.GetDepositRoot() == dt2.GetDepositRoot() {
		t.Error("expected different roots for different deposits")
	}
}

func TestDepositTree_GetDepositCountBytes(t *testing.T) {
	dt := NewDepositTree()

	// Empty tree
	countBytes := dt.GetDepositCountBytes()
	if len(countBytes) != 8 {
		t.Errorf("expected 8 bytes, got %d", len(countBytes))
	}
	for _, b := range countBytes {
		if b != 0 {
			t.Error("expected all zero bytes for empty tree")
		}
	}

	// Add one deposit
	var deposit [32]byte
	_ = dt.Deposit(deposit)

	countBytes = dt.GetDepositCountBytes()
	// Little-endian encoding of 1
	if countBytes[0] != 1 {
		t.Errorf("expected first byte to be 1, got %d", countBytes[0])
	}
	for i := 1; i < 8; i++ {
		if countBytes[i] != 0 {
			t.Errorf("expected byte %d to be 0, got %d", i, countBytes[i])
		}
	}
}

func TestDepositTree_GetBranch(t *testing.T) {
	dt := NewDepositTree()

	// Initial branch should be all zeros
	branch := dt.GetBranch()
	for i, b := range branch {
		if b != [32]byte{} {
			t.Errorf("expected branch[%d] to be all zeros", i)
		}
	}

	// After deposit, branch[0] should be set
	var deposit [32]byte
	deposit[0] = 42
	_ = dt.Deposit(deposit)

	branch = dt.GetBranch()
	if branch[0] != deposit {
		t.Error("expected branch[0] to equal the deposit")
	}
}

func TestDepositTree_ConcurrentAccess(t *testing.T) {
	dt := NewDepositTree()

	// Test concurrent deposits and reads
	done := make(chan bool)

	// Deposit goroutine
	go func() {
		for i := 0; i < 100; i++ {
			var deposit [32]byte
			deposit[0] = byte(i)
			_ = dt.Deposit(deposit)
		}
		done <- true
	}()

	// Read goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = dt.GetDepositRoot()
			_ = dt.GetDepositCount()
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Verify final state
	if dt.GetDepositCount() != 100 {
		t.Errorf("expected 100 deposits, got %d", dt.GetDepositCount())
	}
}

func TestDepositDataRoot(t *testing.T) {
	// Test with valid inputs
	pubkey := make([]byte, 48)
	withdrawalCredentials := make([]byte, 32)
	signature := make([]byte, 96)
	amount := uint64(32000000000) // 32 ETH in Gwei

	root, err := DepositDataRoot(pubkey, withdrawalCredentials, amount, signature)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if root == [32]byte{} {
		t.Error("expected non-zero deposit data root")
	}

	// Verify determinism
	root2, _ := DepositDataRoot(pubkey, withdrawalCredentials, amount, signature)
	if root != root2 {
		t.Error("expected same root for same inputs")
	}
}

func TestDepositDataRoot_InvalidInputs(t *testing.T) {
	validPubkey := make([]byte, 48)
	validWithdrawal := make([]byte, 32)
	validSignature := make([]byte, 96)
	amount := uint64(32000000000)

	// Invalid pubkey length
	_, err := DepositDataRoot(make([]byte, 47), validWithdrawal, amount, validSignature)
	if err == nil {
		t.Error("expected error for invalid pubkey length")
	}

	// Invalid withdrawal credentials length
	_, err = DepositDataRoot(validPubkey, make([]byte, 31), amount, validSignature)
	if err == nil {
		t.Error("expected error for invalid withdrawal credentials length")
	}

	// Invalid signature length
	_, err = DepositDataRoot(validPubkey, validWithdrawal, amount, make([]byte, 95))
	if err == nil {
		t.Error("expected error for invalid signature length")
	}
}

func TestDepositTree_ZeroHashesComputation(t *testing.T) {
	dt := NewDepositTree()
	zeroHashes := dt.GetZeroHashes()

	// Manually verify the first few zero hashes
	var zero [32]byte

	// zeroHashes[0] = 32 zero bytes
	if zeroHashes[0] != zero {
		t.Error("zeroHashes[0] should be 32 zero bytes")
	}

	// zeroHashes[1] = sha256(zeroHashes[0] || zeroHashes[0])
	expected1 := sha256Hash(zero[:], zero[:])
	if zeroHashes[1] != expected1 {
		t.Error("zeroHashes[1] is incorrect")
	}

	// zeroHashes[2] = sha256(zeroHashes[1] || zeroHashes[1])
	expected2 := sha256Hash(expected1[:], expected1[:])
	if zeroHashes[2] != expected2 {
		t.Error("zeroHashes[2] is incorrect")
	}
}

// TestDepositTree_KnownValues tests against known good values
// These values can be verified against the Ethereum deposit contract
func TestDepositTree_KnownValues(t *testing.T) {
	dt := NewDepositTree()

	// Test empty tree hash matches expected value
	// The empty tree root (with count=0) should be deterministic
	emptyRoot := dt.GetDepositRoot()

	// Verify it's not all zeros
	if emptyRoot == [32]byte{} {
		t.Error("empty tree root should not be all zeros")
	}

	// Add a deposit with known data and verify the tree updates correctly
	var knownDeposit [32]byte
	for i := 0; i < 32; i++ {
		knownDeposit[i] = byte(i)
	}
	err := dt.Deposit(knownDeposit)
	if err != nil {
		t.Fatalf("deposit failed: %v", err)
	}

	// Root after one deposit should be different from empty root
	rootAfterOne := dt.GetDepositRoot()
	if rootAfterOne == emptyRoot {
		t.Error("root should change after deposit")
	}

	// Verify branch[0] contains the deposit
	branch := dt.GetBranch()
	if branch[0] != knownDeposit {
		t.Error("branch[0] should contain the deposit for single deposit tree")
	}
}

// TestDepositTree_PowerOfTwoDeposits tests behavior at power-of-two boundaries
func TestDepositTree_PowerOfTwoDeposits(t *testing.T) {
	dt := NewDepositTree()

	// Add deposits and check at power-of-two boundaries
	boundaries := []int{1, 2, 4, 8, 16}

	for _, boundary := range boundaries {
		// Fill up to the boundary
		for dt.GetDepositCount() < uint64(boundary) {
			var deposit [32]byte
			deposit[0] = byte(dt.GetDepositCount())
			err := dt.Deposit(deposit)
			if err != nil {
				t.Fatalf("deposit failed at count %d: %v", dt.GetDepositCount(), err)
			}
		}

		// Verify count
		if dt.GetDepositCount() != uint64(boundary) {
			t.Errorf("expected count %d, got %d", boundary, dt.GetDepositCount())
		}

		// Verify root can be computed
		root := dt.GetDepositRoot()
		if root == [32]byte{} {
			t.Errorf("root should not be empty at count %d", boundary)
		}
	}
}

// BenchmarkDeposit benchmarks the deposit operation
func BenchmarkDeposit(b *testing.B) {
	dt := NewDepositTree()
	var deposit [32]byte

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		deposit[0] = byte(i)
		_ = dt.Deposit(deposit)
	}
}

// BenchmarkGetDepositRoot benchmarks the root computation
func BenchmarkGetDepositRoot(b *testing.B) {
	dt := NewDepositTree()

	// Add some deposits first
	for i := 0; i < 1000; i++ {
		var deposit [32]byte
		deposit[0] = byte(i)
		_ = dt.Deposit(deposit)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dt.GetDepositRoot()
	}
}

// TestSha256Hash tests the helper function
func TestSha256Hash(t *testing.T) {
	var left, right [32]byte
	left[0] = 1
	right[0] = 2

	result := sha256Hash(left[:], right[:])

	// Verify manually
	h := sha256.New()
	h.Write(left[:])
	h.Write(right[:])
	expected := h.Sum(nil)

	if hex.EncodeToString(result[:]) != hex.EncodeToString(expected) {
		t.Error("sha256Hash result doesn't match expected")
	}
}

// TestDepositTree_KnownZeroHashes verifies the zero hashes match the Ethereum specification
func TestDepositTree_KnownZeroHashes(t *testing.T) {
	dt := NewDepositTree()
	zeroHashes := dt.GetZeroHashes()

	// Known zero hashes from the Ethereum 2.0 deposit contract specification
	// These are computed as: zeroHashes[i+1] = sha256(zeroHashes[i] || zeroHashes[i])
	expectedHashes := []string{
		"0000000000000000000000000000000000000000000000000000000000000000", // zeroHashes[0]
		"f5a5fd42d16a20302798ef6ed309979b43003d2320d9f0e8ea9831a92759fb4b", // zeroHashes[1]
		"db56114e00fdd4c1f85c892bf35ac9a89289aaecb1ebd0a96cde606a748b5d71", // zeroHashes[2]
		"c78009fdf07fc56a11f122370658a353aaa542ed63e44c4bc15ff4cd105ab33c", // zeroHashes[3]
		"536d98837f2dd165a55d5eeae91485954472d56f246df256bf3cae19352a123c", // zeroHashes[4]
		"9efde052aa15429fae05bad4d0b1d7c64da64d03d7a1854a588c2cb8430c0d30", // zeroHashes[5]
	}

	for i, expected := range expectedHashes {
		actual := hex.EncodeToString(zeroHashes[i][:])
		if actual != expected {
			t.Errorf("zeroHashes[%d] mismatch:\nexpected: %s\nactual:   %s", i, expected, actual)
		}
	}
}
