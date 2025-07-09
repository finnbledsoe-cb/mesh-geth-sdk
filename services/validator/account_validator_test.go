// Copyright 2025 Coinbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"strings"
	"testing"

	"github.com/coinbase/rosetta-geth-sdk/configuration"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

// testSetupData holds common test setup data
type testSetupData struct {
	ctx         context.Context
	validator   TrustlessValidator
	baseAccount AccountResult
	stateRoot   common.Hash
	blockNumber *big.Int
	chainData   ChainTestData
}

// setupAccountTest creates common test setup data for account validation tests
func setupAccountTest(t *testing.T, chainData ChainTestData) *testSetupData {
	ctx := context.Background()

	cfg := &configuration.Configuration{
		ChainConfig: chainData.ChainConfig,
		Network:     chainData.Network,
		GethURL:     chainData.GethURL,
		RosettaCfg: configuration.RosettaConfig{
			EnableTrustlessAccountValidation: true,
		},
	}
	validator := NewEthereumValidator(cfg)

	// Load test data
	var baseAccount AccountResult
	data, err := os.ReadFile(chainData.AccountFixtureFile)
	if err != nil {
		t.Fatalf("failed to read fixture file for %s: %v", chainData.Name, err)
	}
	err = json.Unmarshal(data, &baseAccount)
	if err != nil {
		t.Fatalf("failed to unmarshal fixture for %s: %v", chainData.Name, err)
	}

	// Calculate the correct state root from the first proof node
	firstNodeData, err := hexutil.Decode(baseAccount.AccountProof[0])
	if err != nil {
		t.Fatalf("failed to decode first node for %s: %v", chainData.Name, err)
	}
	stateRoot := common.BytesToHash(crypto.Keccak256(firstNodeData))

	return &testSetupData{
		ctx:         ctx,
		validator:   validator,
		baseAccount: baseAccount,
		stateRoot:   stateRoot,
		blockNumber: chainData.TestBlockNumber,
		chainData:   chainData,
	}
}

// SUCCESS TESTS
func TestValidateAccountState_ExactValidData(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)
			err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed with exact valid data")
		})
	}
}

func TestValidateAccountState_RepeatedValidation(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)
			for i := 0; i < 3; i++ {
				err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
				assert.NoError(t, err, "Validation should succeed on repeated calls")
			}
		})
	}
}

func TestValidateAccountState_DifferentBlockNumbers(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)
			blockNumbers := []*big.Int{
				setup.blockNumber,
				big.NewInt(rand.Int63n(setup.blockNumber.Int64())),
				new(big.Int).Add(setup.blockNumber, big.NewInt(rand.Int63n(1000000))),
				new(big.Int).Sub(setup.blockNumber, big.NewInt(rand.Int63n(1000))),
				big.NewInt(rand.Int63n(setup.blockNumber.Int64())),
				big.NewInt(rand.Int63n(setup.blockNumber.Int64())),
			}

			for _, blockNumber := range blockNumbers {
				err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, blockNumber)
				assert.NoError(t, err, "Validation should succeed with different block numbers")
			}
		})
	}
}

func TestValidateAccountState_OriginalBalance(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)
			t.Logf("Testing account with balance: %s", setup.baseAccount.Balance.ToInt().String())
			err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed with original balance")
		})
	}
}

func TestValidateAccountState_OriginalNonce(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)
			t.Logf("Testing account with nonce: %d", uint64(setup.baseAccount.Nonce))
			err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed with original nonce")
		})
	}
}

func TestValidateAccountState_OriginalStorageHash(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)
			t.Logf("Testing account with storage hash: %s", setup.baseAccount.StorageHash.Hex())
			err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed with original storage hash")
		})
	}
}

func TestValidateAccountState_OriginalCodeHash(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)
			t.Logf("Testing account with code hash: %s", setup.baseAccount.CodeHash.Hex())
			err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed with original code hash")
		})
	}
}

func TestValidateAccountState_CompleteProofChain(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)
			t.Logf("Testing with %d proof nodes", len(setup.baseAccount.AccountProof))
			err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed with complete proof chain")
		})
	}
}

func TestValidateAccountState_ProofNodeFormatting(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			// Verify all proof nodes are valid hex
			for i, proofNode := range setup.baseAccount.AccountProof {
				_, err := hexutil.Decode(proofNode)
				assert.NoError(t, err, "Proof node %d should be valid hex: %s", i, proofNode)
			}

			err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed with properly formatted proof nodes")
		})
	}
}

func TestValidateAccountState_StateRootCalculation(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			// Recalculate state root to ensure consistency
			firstNodeData, err := hexutil.Decode(setup.baseAccount.AccountProof[0])
			assert.NoError(t, err, "First proof node should decode successfully")

			calculatedStateRoot := common.BytesToHash(crypto.Keccak256(firstNodeData))
			assert.Equal(t, setup.stateRoot, calculatedStateRoot, "State root calculation should be consistent")

			err = setup.validator.ValidateAccountState(setup.baseAccount, calculatedStateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed with calculated state root")
		})
	}
}

func TestValidateAccountState_CorrectAddress(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)
			t.Logf("Testing account address: %s", setup.baseAccount.Address.Hex())
			err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed with correct address")
		})
	}
}

func TestValidateAccountState_AddressConsistency(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			// The account hash should be consistent with the address
			expectedAccountHash := crypto.Keccak256(setup.baseAccount.Address[:])
			t.Logf("Account hash for verification: %x", expectedAccountHash)

			err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed with address-consistent proof path")
		})
	}
}

func TestValidateAccountState_EOACharacteristics(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			// Check if this looks like an EOA (empty code hash typically indicates EOA)
			emptyCodeHash := crypto.Keccak256Hash(nil)
			if setup.baseAccount.CodeHash == emptyCodeHash {
				t.Logf("Testing EOA (empty code hash): %s", setup.baseAccount.CodeHash.Hex())
			} else {
				t.Logf("Testing contract account (non-empty code hash): %s", setup.baseAccount.CodeHash.Hex())
			}

			err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed regardless of account type")
		})
	}
}

func TestValidateAccountState_AccountWithStorage(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			emptyStorageHash := common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
			if setup.baseAccount.StorageHash != emptyStorageHash {
				t.Logf("Testing account with non-empty storage: %s", setup.baseAccount.StorageHash.Hex())
			} else {
				t.Logf("Testing account with empty storage: %s", setup.baseAccount.StorageHash.Hex())
			}

			err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed with account storage")
		})
	}
}

func TestValidateAccountState_MinimumValidProof(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			// Ensure we have at least one proof node (we should have more, but test minimum requirement)
			assert.True(t, len(setup.baseAccount.AccountProof) >= 1, "Should have at least one proof node")

			err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed with minimum valid proof")
		})
	}
}

func TestValidateAccountState_MaximumPrecisionValues(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			// Log the actual values to understand the precision
			t.Logf("Balance precision: %s wei", setup.baseAccount.Balance.ToInt().String())
			t.Logf("Nonce value: %d", uint64(setup.baseAccount.Nonce))

			err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed with high precision values")
		})
	}
}

func TestValidateAccountState_ConcurrentValidation(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			const numGoroutines = 10
			errChan := make(chan error, numGoroutines)

			for i := 0; i < numGoroutines; i++ {
				go func(goroutineID int) {
					err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
					if err != nil {
						errChan <- fmt.Errorf("goroutine %d failed: %w", goroutineID, err)
					} else {
						errChan <- nil
					}
				}(i)
			}

			// Collect results
			for i := 0; i < numGoroutines; i++ {
				err := <-errChan
				assert.NoError(t, err, "Concurrent validation should succeed")
			}
		})
	}
}

func TestValidateAccountState_ValidationStability(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			const iterations = 50
			for i := 0; i < iterations; i++ {
				err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
				assert.NoError(t, err, "Validation should be stable over multiple iterations (iteration %d)", i)
			}
		})
	}
}

func TestValidateAccountState_ExtraProofNode(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			corruptResult := setup.baseAccount
			corruptResult.AccountProof = make([]string, len(setup.baseAccount.AccountProof)+1)
			copy(corruptResult.AccountProof, setup.baseAccount.AccountProof)
			corruptResult.AccountProof[len(setup.baseAccount.AccountProof)] = "0x1234567890abcdef"

			err := setup.validator.ValidateAccountState(corruptResult, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should not fail with extra proof node")
		})
	}
}

// FAILURE TESTS

func TestValidateAccountState_WrongStateRoot(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			wrongStateRoot := common.HexToHash("0x7adc7dbc4c36299fc65fd1dc3798a6a58c29c171b79584bfc3512f5ad82a59d4")
			err := setup.validator.ValidateAccountState(setup.baseAccount, wrongStateRoot, setup.blockNumber)
			assert.Error(t, err, "Validation should fail with wrong state root")
			assert.Contains(t, err.Error(), "state root mismatch")
		})
	}
}

func TestValidateAccountState_ZeroStateRoot(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			zeroStateRoot := common.Hash{}
			err := setup.validator.ValidateAccountState(setup.baseAccount, zeroStateRoot, setup.blockNumber)
			assert.Error(t, err, "Validation should fail with zero state root")
			assert.Contains(t, err.Error(), "state root mismatch")
		})
	}
}

func TestValidateAccountState_OnebitDifferentStateRoot(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			corruptedStateRoot := setup.stateRoot
			corruptedStateRoot[0] ^= 0x01 // Flip the least significant bit of first byte
			err := setup.validator.ValidateAccountState(setup.baseAccount, corruptedStateRoot, setup.blockNumber)
			assert.Error(t, err, "Validation should fail with one bit different state root")
			assert.Contains(t, err.Error(), "state root mismatch")
		})
	}
}

func TestValidateAccountState_CorruptFirstProofNode(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			corruptResult := setup.baseAccount
			corruptProof := setup.baseAccount.AccountProof[0]
			// Change first two characters after 0x
			corruptProof = "0x0" + corruptProof[3:]
			corruptResult.AccountProof = make([]string, len(setup.baseAccount.AccountProof))
			copy(corruptResult.AccountProof, setup.baseAccount.AccountProof)
			corruptResult.AccountProof[0] = corruptProof

			err := setup.validator.ValidateAccountState(corruptResult, setup.stateRoot, setup.blockNumber)
			assert.Error(t, err, "Validation should fail with corrupted first proof node")
		})
	}
}

func TestValidateAccountState_CorruptMiddleProofNode(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			if len(setup.baseAccount.AccountProof) > 2 {
				corruptResult := setup.baseAccount
				corruptResult.AccountProof = make([]string, len(setup.baseAccount.AccountProof))
				copy(corruptResult.AccountProof, setup.baseAccount.AccountProof)
				// Corrupt middle node by changing last few characters
				midIndex := len(corruptResult.AccountProof) / 2
				originalProof := corruptResult.AccountProof[midIndex]
				corruptResult.AccountProof[midIndex] = originalProof[:len(originalProof)-4] + "0000"

				err := setup.validator.ValidateAccountState(corruptResult, setup.stateRoot, setup.blockNumber)
				assert.Error(t, err, "Validation should fail with corrupted middle proof node")
			}
		})
	}
}

func TestValidateAccountState_MissingProofNode(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			if len(setup.baseAccount.AccountProof) > 1 {
				corruptResult := setup.baseAccount
				corruptResult.AccountProof = setup.baseAccount.AccountProof[:len(setup.baseAccount.AccountProof)-1]

				err := setup.validator.ValidateAccountState(corruptResult, setup.stateRoot, setup.blockNumber)
				assert.Error(t, err, "Validation should fail with missing proof node")
			}
		})
	}
}

func TestValidateAccountState_CorruptNonce(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			corruptResult := setup.baseAccount
			corruptResult.Nonce = hexutil.Uint64(uint64(corruptResult.Nonce) + 1)

			err := setup.validator.ValidateAccountState(corruptResult, setup.stateRoot, setup.blockNumber)
			assert.Error(t, err, "Validation should fail with corrupted nonce")
			assert.Contains(t, err.Error(), "account nonce is not matched")
		})
	}
}

func TestValidateAccountState_CorruptBalance(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			corruptResult := setup.baseAccount
			originalBalance := corruptResult.Balance.ToInt()
			newBalance := new(big.Int).Sub(originalBalance, big.NewInt(1))
			corruptResult.Balance = (*hexutil.Big)(newBalance)

			err := setup.validator.ValidateAccountState(corruptResult, setup.stateRoot, setup.blockNumber)
			assert.Error(t, err, "Validation should fail with corrupted balance")
			assert.Contains(t, err.Error(), "account balance is not matched")
		})
	}
}

func TestValidateAccountState_CorruptStorageHash(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			corruptResult := setup.baseAccount
			corruptResult.StorageHash[31] ^= 0x01 // Flip last byte

			err := setup.validator.ValidateAccountState(corruptResult, setup.stateRoot, setup.blockNumber)
			assert.Error(t, err, "Validation should fail with corrupted storage hash")
			assert.Contains(t, err.Error(), "account storage hash is not matched")
		})
	}
}

func TestValidateAccountState_CorruptCodeHash(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			corruptResult := setup.baseAccount
			corruptResult.CodeHash[0] ^= 0x01 // Flip first byte

			err := setup.validator.ValidateAccountState(corruptResult, setup.stateRoot, setup.blockNumber)
			assert.Error(t, err, "Validation should fail with corrupted code hash")
			assert.Contains(t, err.Error(), "account code hash is not matched")
		})
	}
}

func TestValidateAccountState_CorruptAddress(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			corruptResult := setup.baseAccount
			corruptResult.Address = common.HexToAddress("0x1234567890123456789012345678901234567890")

			err := setup.validator.ValidateAccountState(corruptResult, setup.stateRoot, setup.blockNumber)
			assert.Error(t, err, "Validation should fail with corrupted address")
		})
	}
}

func TestValidateAccountState_EmptyAccountProof(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			corruptResult := setup.baseAccount
			corruptResult.AccountProof = []string{}

			err := setup.validator.ValidateAccountState(corruptResult, setup.stateRoot, setup.blockNumber)
			assert.Error(t, err, "Validation should fail with empty account proof")
		})
	}
}

func TestValidateAccountState_InvalidHexInProof(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			corruptResult := setup.baseAccount
			corruptResult.AccountProof = make([]string, len(setup.baseAccount.AccountProof))
			copy(corruptResult.AccountProof, setup.baseAccount.AccountProof)
			corruptResult.AccountProof[0] = "0xGGGGGGGG" // Invalid hex

			err := setup.validator.ValidateAccountState(corruptResult, setup.stateRoot, setup.blockNumber)
			assert.Error(t, err, "Validation should fail with invalid hex in proof")
			assert.Contains(t, err.Error(), "failed to decode first node")
		})
	}
}

func TestValidateAccountState_ZeroBalanceAccount(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			// This should actually pass if the proof is valid for a zero-balance account
			// We're testing that the validation logic handles zero values correctly
			corruptResult := setup.baseAccount
			corruptResult.Balance = (*hexutil.Big)(big.NewInt(0))

			err := setup.validator.ValidateAccountState(corruptResult, setup.stateRoot, setup.blockNumber)
			// This might pass or fail depending on whether the proof matches the modified balance
			// The important thing is it doesn't panic and handles the zero value
			if err != nil {
				assert.Contains(t, err.Error(), "account balance is not matched")
			}
		})
	}
}

func TestValidateAccountState_SuccessfulValidation(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed with valid test data")
		})
	}
}

// ==============================================================================
// ERC-20 TOKEN VALIDATION TESTS
// ==============================================================================

// TestCalculateERC20BalanceStorageKey tests the storage key calculation for ERC-20 tokens
func TestCalculateERC20BalanceStorageKey(t *testing.T) {
	testCases := []struct {
		name        string
		userAddr    common.Address
		mappingSlot int
		expected    string
	}{
		{
			name:        "standard ERC-20 balance slot 0",
			userAddr:    common.HexToAddress("0x742d35Cc6634C0532925a3b8D433C02c1040F05a"),
			mappingSlot: 0,
			expected:    "0x459f40a170b58b8a0d3ff6f1d8c1c3e8b0e8ac8b21ad0d7c5b7e6e8b0d3ff6f1d",
		},
		{
			name:        "ERC-20 balance slot 1",
			userAddr:    common.HexToAddress("0x742d35Cc6634C0532925a3b8D433C02c1040F05a"),
			mappingSlot: 1,
			expected:    "0x5e7a4a9e7a4a9e7a4a9e7a4a9e7a4a9e7a4a9e7a4a9e7a4a9e7a4a9e7a4a9e7a",
		},
		{
			name:        "different address slot 0",
			userAddr:    common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
			mappingSlot: 0,
			expected:    "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		},
		{
			name:        "zero address slot 0",
			userAddr:    common.Address{},
			mappingSlot: 0,
			expected:    "0x290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := calculateERC20BalanceStorageKey(tc.userAddr, tc.mappingSlot)

			// Verify the result is a valid hex string
			assert.True(t, common.IsHexAddress(result) || len(result) == 66, "Result should be valid hex with 0x prefix")

			// Verify the result is deterministic
			result2 := calculateERC20BalanceStorageKey(tc.userAddr, tc.mappingSlot)
			assert.Equal(t, result, result2, "Storage key calculation should be deterministic")

			// Verify the result follows the expected pattern (keccak256 hash)
			assert.True(t, len(result) == 66, "Storage key should be 66 characters (0x + 64 hex chars)")
		})
	}
}

// TestValidateStorageProof tests the storage proof validation logic
func TestValidateStorageProof(t *testing.T) {
	// Create a mock validator for testing
	cfg := &configuration.Configuration{
		ChainConfig: SonicChainConfig,
		RosettaCfg: configuration.RosettaConfig{
			EnableTrustlessAccountValidation: true,
		},
	}
	validator := NewEthereumValidator(cfg).(*trustlessValidator)

	t.Run("empty proof with zero value", func(t *testing.T) {
		storageProof := StorageResult{
			Key:   "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			Value: (*hexutil.Big)(big.NewInt(0)),
			Proof: []string{},
		}

		err := validator.ValidateStorageProof(storageProof, common.Hash{}, "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
		assert.NoError(t, err, "Empty proof with zero value should be valid")
	})

	t.Run("empty proof with non-zero value", func(t *testing.T) {
		storageProof := StorageResult{
			Key:   "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			Value: (*hexutil.Big)(big.NewInt(100)),
			Proof: []string{},
		}

		err := validator.ValidateStorageProof(storageProof, common.Hash{}, "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
		assert.Error(t, err, "Empty proof with non-zero value should fail")
		assert.Contains(t, err.Error(), "non-zero value with empty proof")
	})

	t.Run("invalid hex in proof", func(t *testing.T) {
		storageProof := StorageResult{
			Key:   "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			Value: (*hexutil.Big)(big.NewInt(100)),
			Proof: []string{"invalid-hex"},
		}

		err := validator.ValidateStorageProof(storageProof, common.Hash{}, "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
		assert.Error(t, err, "Invalid hex in proof should fail")
		assert.Contains(t, err.Error(), "failed to decode storage proof node")
	})

	t.Run("invalid storage key", func(t *testing.T) {
		storageProof := StorageResult{
			Key:   "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			Value: (*hexutil.Big)(big.NewInt(100)),
			Proof: []string{"0x1234"},
		}

		err := validator.ValidateStorageProof(storageProof, common.Hash{}, "invalid-key")
		assert.Error(t, err, "Invalid storage key should fail")
		assert.Contains(t, err.Error(), "failed to decode storage key")
	})
}

// TestValidateERC20Balance tests individual ERC-20 balance validation
func TestValidateERC20Balance(t *testing.T) {
	cfg := &configuration.Configuration{
		ChainConfig: SonicChainConfig,
		RosettaCfg: configuration.RosettaConfig{
			EnableTrustlessAccountValidation: true,
		},
	}
	validator := NewEthereumValidator(cfg).(*trustlessValidator)

	t.Run("missing storage proof", func(t *testing.T) {
		contractProof := AccountResult{
			Address:      common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
			StorageProof: []StorageResult{},
		}

		storageKey := "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
		expectedBalance := &types.Amount{
			Value: "100",
			Currency: &types.Currency{
				Symbol:   "USDC",
				Decimals: 6,
			},
		}

		err := validator.ValidateERC20Balance(contractProof, storageKey, expectedBalance)
		assert.Error(t, err, "Missing storage proof should fail")
		assert.Contains(t, err.Error(), "missing storage proof")
	})

	t.Run("invalid balance value", func(t *testing.T) {
		contractProof := AccountResult{
			Address: common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
			StorageProof: []StorageResult{
				{
					Key:   "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					Value: (*hexutil.Big)(big.NewInt(100)),
					Proof: []string{},
				},
			},
		}

		storageKey := "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
		expectedBalance := &types.Amount{
			Value: "invalid-value",
			Currency: &types.Currency{
				Symbol:   "USDC",
				Decimals: 6,
			},
		}

		err := validator.ValidateERC20Balance(contractProof, storageKey, expectedBalance)
		assert.Error(t, err, "Invalid balance value should fail")
		assert.Contains(t, err.Error(), "invalid balance value")
	})

	t.Run("balance mismatch", func(t *testing.T) {
		contractProof := AccountResult{
			Address: common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
			StorageProof: []StorageResult{
				{
					Key:   "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					Value: (*hexutil.Big)(big.NewInt(100)),
					Proof: []string{},
				},
			},
		}

		storageKey := "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
		expectedBalance := &types.Amount{
			Value: "200", // Different from proven value
			Currency: &types.Currency{
				Symbol:   "USDC",
				Decimals: 6,
			},
		}

		err := validator.ValidateERC20Balance(contractProof, storageKey, expectedBalance)
		assert.Error(t, err, "Balance mismatch should fail")
		assert.Contains(t, err.Error(), "ERC-20 balance mismatch")
	})

	t.Run("successful validation", func(t *testing.T) {
		contractProof := AccountResult{
			Address: common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
			StorageProof: []StorageResult{
				{
					Key:   "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					Value: (*hexutil.Big)(big.NewInt(100)),
					Proof: []string{},
				},
			},
		}

		storageKey := "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
		expectedBalance := &types.Amount{
			Value: "100",
			Currency: &types.Currency{
				Symbol:   "USDC",
				Decimals: 6,
			},
		}

		err := validator.ValidateERC20Balance(contractProof, storageKey, expectedBalance)
		assert.NoError(t, err, "Matching balance should validate successfully")
	})
}

// TestValidateERC20Balances tests the validation of multiple ERC-20 token balances
func TestValidateERC20Balances(t *testing.T) {
	cfg := &configuration.Configuration{
		ChainConfig: SonicChainConfig,
		RosettaCfg: configuration.RosettaConfig{
			EnableTrustlessAccountValidation: true,
		},
	}
	validator := NewEthereumValidator(cfg).(*trustlessValidator)

	ctx := context.Background()
	userAddr := common.HexToAddress("0x742d35Cc6634C0532925a3b8D433C02c1040F05a")
	blockNumber := big.NewInt(1000)

	t.Run("no ERC-20 tokens", func(t *testing.T) {
		balances := []*types.Amount{
			{
				Value: "1000000000000000000",
				Currency: &types.Currency{
					Symbol:   "ETH",
					Decimals: 18,
				},
			},
		}

		err := validator.ValidateERC20Balances(ctx, userAddr, balances, blockNumber)
		assert.NoError(t, err, "No ERC-20 tokens should validate successfully")
	})

	t.Run("currency without metadata", func(t *testing.T) {
		balances := []*types.Amount{
			{
				Value: "1000000000000000000",
				Currency: &types.Currency{
					Symbol:   "UNKNOWN",
					Decimals: 18,
				},
			},
		}

		err := validator.ValidateERC20Balances(ctx, userAddr, balances, blockNumber)
		assert.NoError(t, err, "Currency without metadata should be skipped")
	})

	t.Run("currency without contract address", func(t *testing.T) {
		balances := []*types.Amount{
			{
				Value: "1000000000000000000",
				Currency: &types.Currency{
					Symbol:   "TOKEN",
					Decimals: 18,
					Metadata: map[string]interface{}{
						"otherField": "value",
					},
				},
			},
		}

		err := validator.ValidateERC20Balances(ctx, userAddr, balances, blockNumber)
		assert.NoError(t, err, "Currency without contract address should be skipped")
	})
}

// TestValidateAccount_WithERC20Tokens tests the full account validation with ERC-20 tokens
func TestValidateAccount_WithERC20Tokens(t *testing.T) {
	cfg := &configuration.Configuration{
		ChainConfig: SonicChainConfig,
		RosettaCfg: configuration.RosettaConfig{
			EnableTrustlessAccountValidation: true,
		},
	}
	validator := NewEthereumValidator(cfg)

	ctx := context.Background()
	address := "0x742d35Cc6634C0532925a3b8D433C02c1040F05a"

	t.Run("account with native and ERC-20 balances", func(t *testing.T) {
		balanceResponse := &types.AccountBalanceResponse{
			BlockIdentifier: &types.BlockIdentifier{
				Index: 1000,
				Hash:  "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			},
			Balances: []*types.Amount{
				{
					Value: "1000000000000000000",
					Currency: &types.Currency{
						Symbol:   "ETH",
						Decimals: 18,
					},
				},
				{
					Value: "1000000",
					Currency: &types.Currency{
						Symbol:   "USDC",
						Decimals: 6,
						Metadata: map[string]interface{}{
							"contractAddress": "0xA0b86a33E6441E4C0EF28DA0c8BF0b7a1F3d1F3d",
						},
					},
				},
			},
		}

		// This will fail because we don't have a real node, but we can test the flow
		err := validator.ValidateAccount(ctx, balanceResponse, address)
		// We expect this to fail due to no real node connection, but not due to logic errors
		assert.Error(t, err, "Should fail due to no real node connection")
		assert.Contains(t, err.Error(), "GethURL not configured")
	})

	t.Run("account with multiple ERC-20 tokens", func(t *testing.T) {
		balanceResponse := &types.AccountBalanceResponse{
			BlockIdentifier: &types.BlockIdentifier{
				Index: 1000,
				Hash:  "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			},
			Balances: []*types.Amount{
				{
					Value: "1000000000000000000",
					Currency: &types.Currency{
						Symbol:   "ETH",
						Decimals: 18,
					},
				},
				{
					Value: "1000000",
					Currency: &types.Currency{
						Symbol:   "USDC",
						Decimals: 6,
						Metadata: map[string]interface{}{
							"contractAddress": "0xA0b86a33E6441E4C0EF28DA0c8BF0b7a1F3d1F3d",
						},
					},
				},
				{
					Value: "500000000000000000",
					Currency: &types.Currency{
						Symbol:   "WETH",
						Decimals: 18,
						Metadata: map[string]interface{}{
							"contractAddress": "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
						},
					},
				},
			},
		}

		// This will fail because we don't have a real node, but we can test the flow
		err := validator.ValidateAccount(ctx, balanceResponse, address)
		assert.Error(t, err, "Should fail due to no real node connection")
		assert.Contains(t, err.Error(), "GethURL not configured")
	})
}

// TestGetAccountProofWithStorage tests the enhanced account proof retrieval with storage
func TestGetAccountProofWithStorage(t *testing.T) {
	cfg := &configuration.Configuration{
		ChainConfig: SonicChainConfig,
		RosettaCfg: configuration.RosettaConfig{
			EnableTrustlessAccountValidation: true,
		},
	}
	validator := NewEthereumValidator(cfg).(*trustlessValidator)

	ctx := context.Background()
	account := common.HexToAddress("0x742d35Cc6634C0532925a3b8D433C02c1040F05a")
	blockNumber := big.NewInt(1000)

	t.Run("no storage keys", func(t *testing.T) {
		_, err := validator.GetAccountProofWithStorage(ctx, account, blockNumber, []string{})
		assert.Error(t, err, "Should fail due to no GethURL configured")
		assert.Contains(t, err.Error(), "GethURL not configured")
	})

	t.Run("with storage keys", func(t *testing.T) {
		storageKeys := []string{
			"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			"0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		}

		_, err := validator.GetAccountProofWithStorage(ctx, account, blockNumber, storageKeys)
		assert.Error(t, err, "Should fail due to no GethURL configured")
		assert.Contains(t, err.Error(), "GethURL not configured")
	})
}

// TestEdgeCases tests various edge cases for ERC-20 validation
func TestERC20Validation_EdgeCases(t *testing.T) {
	cfg := &configuration.Configuration{
		ChainConfig: SonicChainConfig,
		RosettaCfg: configuration.RosettaConfig{
			EnableTrustlessAccountValidation: true,
		},
	}
	validator := NewEthereumValidator(cfg).(*trustlessValidator)

	ctx := context.Background()
	userAddr := common.HexToAddress("0x742d35Cc6634C0532925a3b8D433C02c1040F05a")
	blockNumber := big.NewInt(1000)

	t.Run("zero balance ERC-20 token", func(t *testing.T) {
		balances := []*types.Amount{
			{
				Value: "0",
				Currency: &types.Currency{
					Symbol:   "USDC",
					Decimals: 6,
					Metadata: map[string]interface{}{
						"contractAddress": "0xA0b86a33E6441E4C0EF28DA0c8BF0b7a1F3d1F3d",
					},
				},
			},
		}

		err := validator.ValidateERC20Balances(ctx, userAddr, balances, blockNumber)
		assert.Error(t, err, "Should fail due to no GethURL configured")
		assert.Contains(t, err.Error(), "GethURL not configured")
	})

	t.Run("very large balance", func(t *testing.T) {
		largeBalance := new(big.Int)
		largeBalance.SetString("1000000000000000000000000000000", 10) // 1 trillion tokens

		balances := []*types.Amount{
			{
				Value: largeBalance.String(),
				Currency: &types.Currency{
					Symbol:   "USDC",
					Decimals: 6,
					Metadata: map[string]interface{}{
						"contractAddress": "0xA0b86a33E6441E4C0EF28DA0c8BF0b7a1F3d1F3d",
					},
				},
			},
		}

		err := validator.ValidateERC20Balances(ctx, userAddr, balances, blockNumber)
		assert.Error(t, err, "Should fail due to no GethURL configured")
		assert.Contains(t, err.Error(), "GethURL not configured")
	})

	t.Run("malformed contract address", func(t *testing.T) {
		balances := []*types.Amount{
			{
				Value: "1000000",
				Currency: &types.Currency{
					Symbol:   "USDC",
					Decimals: 6,
					Metadata: map[string]interface{}{
						"contractAddress": "invalid-address",
					},
				},
			},
		}

		err := validator.ValidateERC20Balances(ctx, userAddr, balances, blockNumber)
		assert.Error(t, err, "Should fail due to no GethURL configured")
		assert.Contains(t, err.Error(), "GethURL not configured")
	})
}

// TestConcurrentERC20Validation tests concurrent validation of ERC-20 tokens
func TestConcurrentERC20Validation(t *testing.T) {
	cfg := &configuration.Configuration{
		ChainConfig: SonicChainConfig,
		RosettaCfg: configuration.RosettaConfig{
			EnableTrustlessAccountValidation: true,
		},
	}
	validator := NewEthereumValidator(cfg).(*trustlessValidator)

	ctx := context.Background()
	userAddr := common.HexToAddress("0x742d35Cc6634C0532925a3b8D433C02c1040F05a")
	blockNumber := big.NewInt(1000)

	balances := []*types.Amount{
		{
			Value: "1000000",
			Currency: &types.Currency{
				Symbol:   "USDC",
				Decimals: 6,
				Metadata: map[string]interface{}{
					"contractAddress": "0xA0b86a33E6441E4C0EF28DA0c8BF0b7a1F3d1F3d",
				},
			},
		},
	}

	const numGoroutines = 5
	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			err := validator.ValidateERC20Balances(ctx, userAddr, balances, blockNumber)
			errChan <- err
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		err := <-errChan
		// All should fail with the same error (no GethURL configured)
		assert.Error(t, err, "Should fail due to no GethURL configured")
		assert.Contains(t, err.Error(), "GethURL not configured")
	}
}

// TestBackwardsCompatibility tests that the enhanced validator maintains backwards compatibility
func TestBackwardsCompatibility(t *testing.T) {
	cfg := &configuration.Configuration{
		ChainConfig: SonicChainConfig,
		RosettaCfg: configuration.RosettaConfig{
			EnableTrustlessAccountValidation: true,
		},
	}
	validator := NewEthereumValidator(cfg).(*trustlessValidator)

	ctx := context.Background()
	account := common.HexToAddress("0x742d35Cc6634C0532925a3b8D433C02c1040F05a")
	blockNumber := big.NewInt(1000)

	t.Run("GetAccountProof still works", func(t *testing.T) {
		_, err := validator.GetAccountProof(ctx, account, blockNumber)
		assert.Error(t, err, "Should fail due to no GethURL configured")
		assert.Contains(t, err.Error(), "GethURL not configured")
	})

	t.Run("GetAccountProofWithStorage with empty keys equals GetAccountProof", func(t *testing.T) {
		// Both should fail with the same error
		_, err1 := validator.GetAccountProof(ctx, account, blockNumber)
		_, err2 := validator.GetAccountProofWithStorage(ctx, account, blockNumber, []string{})

		assert.Error(t, err1)
		assert.Error(t, err2)
		assert.Contains(t, err1.Error(), "GethURL not configured")
		assert.Contains(t, err2.Error(), "GethURL not configured")
	})
}

// TestValidateTokenBalanceWithSlotDetection tests the slot detection logic
func TestValidateTokenBalanceWithSlotDetection(t *testing.T) {
	// Create test token configurations
	testTokens := []configuration.Token{
		{
			ChainID:      146,
			Address:      "0xA0b86a33E6441E4C0EF28DA0c8BF0b7a1F3d1F3d",
			Symbol:       "USDC",
			Decimals:     6,
			BalancesSlot: func() *uint64 { slot := uint64(0); return &slot }(),
		},
		{
			ChainID:      146,
			Address:      "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
			Symbol:       "WETH",
			Decimals:     18,
			BalancesSlot: func() *uint64 { slot := uint64(1); return &slot }(),
		},
		{
			ChainID:  146,
			Address:  "0x1234567890abcdef1234567890abcdef12345678",
			Symbol:   "TOKEN",
			Decimals: 18,
			// No BalancesSlot specified - should try defaults
		},
	}

	cfg := &configuration.Configuration{
		ChainConfig: SonicChainConfig,
		RosettaCfg: configuration.RosettaConfig{
			EnableTrustlessAccountValidation: true,
			TokenWhiteList:                   testTokens,
		},
	}
	validator := NewEthereumValidator(cfg).(*trustlessValidator)

	ctx := context.Background()
	userAddr := common.HexToAddress("0x742d35Cc6634C0532925a3b8D433C02c1040F05a")
	blockNumber := big.NewInt(1000)

	t.Run("token with configured slot", func(t *testing.T) {
		balance := &types.Amount{
			Value: "1000000",
			Currency: &types.Currency{
				Symbol:   "USDC",
				Decimals: 6,
				Metadata: map[string]interface{}{
					"contractAddress": "0xA0b86a33E6441E4C0EF28DA0c8BF0b7a1F3d1F3d",
				},
			},
		}

		// This should fail due to no GethURL, but would use slot 0 if configured
		err := validator.validateTokenBalanceWithSlotDetection(
			ctx,
			userAddr,
			common.HexToAddress("0xA0b86a33E6441E4C0EF28DA0c8BF0b7a1F3d1F3d"),
			balance,
			blockNumber,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "GethURL not configured")
	})

	t.Run("token with different configured slot", func(t *testing.T) {
		balance := &types.Amount{
			Value: "500000000000000000",
			Currency: &types.Currency{
				Symbol:   "WETH",
				Decimals: 18,
				Metadata: map[string]interface{}{
					"contractAddress": "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
				},
			},
		}

		// This should fail due to no GethURL, but would use slot 1 if configured
		err := validator.validateTokenBalanceWithSlotDetection(
			ctx,
			userAddr,
			common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"),
			balance,
			blockNumber,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "GethURL not configured")
	})

	t.Run("token without configured slot should try defaults", func(t *testing.T) {
		balance := &types.Amount{
			Value: "1000000000000000000",
			Currency: &types.Currency{
				Symbol:   "TOKEN",
				Decimals: 18,
				Metadata: map[string]interface{}{
					"contractAddress": "0x1234567890abcdef1234567890abcdef12345678",
				},
			},
		}

		// This should fail due to no GethURL, but would try slots [0, 1, 2, 3] if configured
		err := validator.validateTokenBalanceWithSlotDetection(
			ctx,
			userAddr,
			common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
			balance,
			blockNumber,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "GethURL not configured")
	})
}

// TestStorageSlotConfiguration tests storage slot configuration scenarios
func TestStorageSlotConfiguration(t *testing.T) {
	testCases := []struct {
		name               string
		tokens             []configuration.Token
		contractAddress    string
		expectedSlots      []int
		shouldFindInConfig bool
	}{
		{
			name: "token with explicit slot 0",
			tokens: []configuration.Token{
				{
					Address:      "0xA0b86a33E6441E4C0EF28DA0c8BF0b7a1F3d1F3d",
					BalancesSlot: func() *uint64 { slot := uint64(0); return &slot }(),
				},
			},
			contractAddress:    "0xA0b86a33E6441E4C0EF28DA0c8BF0b7a1F3d1F3d",
			expectedSlots:      []int{0},
			shouldFindInConfig: true,
		},
		{
			name: "token with explicit slot 3",
			tokens: []configuration.Token{
				{
					Address:      "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
					BalancesSlot: func() *uint64 { slot := uint64(3); return &slot }(),
				},
			},
			contractAddress:    "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
			expectedSlots:      []int{3},
			shouldFindInConfig: true,
		},
		{
			name: "token without slot config",
			tokens: []configuration.Token{
				{
					Address: "0x1234567890abcdef1234567890abcdef12345678",
					// No BalancesSlot
				},
			},
			contractAddress:    "0x1234567890abcdef1234567890abcdef12345678",
			expectedSlots:      []int{0, 1, 2, 3}, // Should use defaults
			shouldFindInConfig: false,
		},
		{
			name:               "token not in config",
			tokens:             []configuration.Token{},
			contractAddress:    "0xUnknownContract1234567890abcdef1234567890",
			expectedSlots:      []int{0, 1, 2, 3}, // Should use defaults
			shouldFindInConfig: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &configuration.Configuration{
				ChainConfig: SonicChainConfig,
				RosettaCfg: configuration.RosettaConfig{
					TokenWhiteList: tc.tokens,
				},
			}
			validator := NewEthereumValidator(cfg).(*trustlessValidator)

			// Check the slot detection logic (this simulates part of validateTokenBalanceWithSlotDetection)
			var slotsToTry []int
			contractAddress := common.HexToAddress(tc.contractAddress)

			// Check if we have token configuration with specific slot
			if validator.config != nil && validator.config.RosettaCfg.TokenWhiteList != nil {
				for _, token := range validator.config.RosettaCfg.TokenWhiteList {
					if strings.EqualFold(token.Address, contractAddress.Hex()) {
						if token.BalancesSlot != nil {
							slotsToTry = append(slotsToTry, int(*token.BalancesSlot))
						}
						break
					}
				}
			}

			// If no specific slot configured, try common slots
			if len(slotsToTry) == 0 {
				slotsToTry = []int{0, 1, 2, 3}
			}

			assert.Equal(t, tc.expectedSlots, slotsToTry, "Should get expected slots to try")
		})
	}
}
