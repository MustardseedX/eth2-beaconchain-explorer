// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package validator_registration

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// ValidatorRegistrationMetaData contains all meta data concerning the ValidatorRegistration contract.
var ValidatorRegistrationMetaData = &bind.MetaData{
	ABI: "[{\"name\":\"DepositEvent\",\"inputs\":[{\"type\":\"bytes\",\"name\":\"pubkey\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"withdrawal_credentials\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"amount\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"signature\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"index\",\"indexed\":false}],\"anonymous\":false,\"type\":\"event\"},{\"outputs\":[],\"inputs\":[],\"constant\":false,\"payable\":false,\"type\":\"constructor\"},{\"name\":\"get_deposit_root\",\"outputs\":[{\"type\":\"bytes32\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":95628},{\"name\":\"get_deposit_count\",\"outputs\":[{\"type\":\"bytes\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":18231},{\"name\":\"deposit\",\"outputs\":[],\"inputs\":[{\"type\":\"bytes\",\"name\":\"pubkey\"},{\"type\":\"bytes\",\"name\":\"withdrawal_credentials\"},{\"type\":\"bytes\",\"name\":\"signature\"},{\"type\":\"bytes32\",\"name\":\"deposit_data_root\"}],\"constant\":false,\"payable\":true,\"type\":\"function\",\"gas\":1342274}]",
}

// ValidatorRegistrationABI is the input ABI used to generate the binding from.
// Deprecated: Use ValidatorRegistrationMetaData.ABI instead.
var ValidatorRegistrationABI = ValidatorRegistrationMetaData.ABI

// ValidatorRegistration is an auto generated Go binding around an Ethereum contract.
type ValidatorRegistration struct {
	ValidatorRegistrationCaller     // Read-only binding to the contract
	ValidatorRegistrationTransactor // Write-only binding to the contract
	ValidatorRegistrationFilterer   // Log filterer for contract events
}

// ValidatorRegistrationCaller is an auto generated read-only Go binding around an Ethereum contract.
type ValidatorRegistrationCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ValidatorRegistrationTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ValidatorRegistrationTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ValidatorRegistrationFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ValidatorRegistrationFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ValidatorRegistrationSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ValidatorRegistrationSession struct {
	Contract     *ValidatorRegistration // Generic contract binding to set the session for
	CallOpts     bind.CallOpts          // Call options to use throughout this session
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// ValidatorRegistrationCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ValidatorRegistrationCallerSession struct {
	Contract *ValidatorRegistrationCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                // Call options to use throughout this session
}

// ValidatorRegistrationTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ValidatorRegistrationTransactorSession struct {
	Contract     *ValidatorRegistrationTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                // Transaction auth options to use throughout this session
}

// ValidatorRegistrationRaw is an auto generated low-level Go binding around an Ethereum contract.
type ValidatorRegistrationRaw struct {
	Contract *ValidatorRegistration // Generic contract binding to access the raw methods on
}

// ValidatorRegistrationCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ValidatorRegistrationCallerRaw struct {
	Contract *ValidatorRegistrationCaller // Generic read-only contract binding to access the raw methods on
}

// ValidatorRegistrationTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ValidatorRegistrationTransactorRaw struct {
	Contract *ValidatorRegistrationTransactor // Generic write-only contract binding to access the raw methods on
}

// NewValidatorRegistration creates a new instance of ValidatorRegistration, bound to a specific deployed contract.
func NewValidatorRegistration(address common.Address, backend bind.ContractBackend) (*ValidatorRegistration, error) {
	contract, err := bindValidatorRegistration(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ValidatorRegistration{ValidatorRegistrationCaller: ValidatorRegistrationCaller{contract: contract}, ValidatorRegistrationTransactor: ValidatorRegistrationTransactor{contract: contract}, ValidatorRegistrationFilterer: ValidatorRegistrationFilterer{contract: contract}}, nil
}

// NewValidatorRegistrationCaller creates a new read-only instance of ValidatorRegistration, bound to a specific deployed contract.
func NewValidatorRegistrationCaller(address common.Address, caller bind.ContractCaller) (*ValidatorRegistrationCaller, error) {
	contract, err := bindValidatorRegistration(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ValidatorRegistrationCaller{contract: contract}, nil
}

// NewValidatorRegistrationTransactor creates a new write-only instance of ValidatorRegistration, bound to a specific deployed contract.
func NewValidatorRegistrationTransactor(address common.Address, transactor bind.ContractTransactor) (*ValidatorRegistrationTransactor, error) {
	contract, err := bindValidatorRegistration(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ValidatorRegistrationTransactor{contract: contract}, nil
}

// NewValidatorRegistrationFilterer creates a new log filterer instance of ValidatorRegistration, bound to a specific deployed contract.
func NewValidatorRegistrationFilterer(address common.Address, filterer bind.ContractFilterer) (*ValidatorRegistrationFilterer, error) {
	contract, err := bindValidatorRegistration(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ValidatorRegistrationFilterer{contract: contract}, nil
}

// bindValidatorRegistration binds a generic wrapper to an already deployed contract.
func bindValidatorRegistration(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ValidatorRegistrationMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ValidatorRegistration *ValidatorRegistrationRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ValidatorRegistration.Contract.ValidatorRegistrationCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ValidatorRegistration *ValidatorRegistrationRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ValidatorRegistration.Contract.ValidatorRegistrationTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ValidatorRegistration *ValidatorRegistrationRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ValidatorRegistration.Contract.ValidatorRegistrationTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ValidatorRegistration *ValidatorRegistrationCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ValidatorRegistration.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ValidatorRegistration *ValidatorRegistrationTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ValidatorRegistration.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ValidatorRegistration *ValidatorRegistrationTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ValidatorRegistration.Contract.contract.Transact(opts, method, params...)
}

// GetDepositCount is a free data retrieval call binding the contract method 0x621fd130.
//
// Solidity: function get_deposit_count() returns(bytes out)
func (_ValidatorRegistration *ValidatorRegistrationCaller) GetDepositCount(opts *bind.CallOpts) ([]byte, error) {
	var out []interface{}
	err := _ValidatorRegistration.contract.Call(opts, &out, "get_deposit_count")

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// GetDepositCount is a free data retrieval call binding the contract method 0x621fd130.
//
// Solidity: function get_deposit_count() returns(bytes out)
func (_ValidatorRegistration *ValidatorRegistrationSession) GetDepositCount() ([]byte, error) {
	return _ValidatorRegistration.Contract.GetDepositCount(&_ValidatorRegistration.CallOpts)
}

// GetDepositCount is a free data retrieval call binding the contract method 0x621fd130.
//
// Solidity: function get_deposit_count() returns(bytes out)
func (_ValidatorRegistration *ValidatorRegistrationCallerSession) GetDepositCount() ([]byte, error) {
	return _ValidatorRegistration.Contract.GetDepositCount(&_ValidatorRegistration.CallOpts)
}

// GetDepositRoot is a free data retrieval call binding the contract method 0xc5f2892f.
//
// Solidity: function get_deposit_root() returns(bytes32 out)
func (_ValidatorRegistration *ValidatorRegistrationCaller) GetDepositRoot(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _ValidatorRegistration.contract.Call(opts, &out, "get_deposit_root")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetDepositRoot is a free data retrieval call binding the contract method 0xc5f2892f.
//
// Solidity: function get_deposit_root() returns(bytes32 out)
func (_ValidatorRegistration *ValidatorRegistrationSession) GetDepositRoot() ([32]byte, error) {
	return _ValidatorRegistration.Contract.GetDepositRoot(&_ValidatorRegistration.CallOpts)
}

// GetDepositRoot is a free data retrieval call binding the contract method 0xc5f2892f.
//
// Solidity: function get_deposit_root() returns(bytes32 out)
func (_ValidatorRegistration *ValidatorRegistrationCallerSession) GetDepositRoot() ([32]byte, error) {
	return _ValidatorRegistration.Contract.GetDepositRoot(&_ValidatorRegistration.CallOpts)
}

// Deposit is a paid mutator transaction binding the contract method 0x22895118.
//
// Solidity: function deposit(bytes pubkey, bytes withdrawal_credentials, bytes signature, bytes32 deposit_data_root) returns()
func (_ValidatorRegistration *ValidatorRegistrationTransactor) Deposit(opts *bind.TransactOpts, pubkey []byte, withdrawal_credentials []byte, signature []byte, deposit_data_root [32]byte) (*types.Transaction, error) {
	return _ValidatorRegistration.contract.Transact(opts, "deposit", pubkey, withdrawal_credentials, signature, deposit_data_root)
}

// Deposit is a paid mutator transaction binding the contract method 0x22895118.
//
// Solidity: function deposit(bytes pubkey, bytes withdrawal_credentials, bytes signature, bytes32 deposit_data_root) returns()
func (_ValidatorRegistration *ValidatorRegistrationSession) Deposit(pubkey []byte, withdrawal_credentials []byte, signature []byte, deposit_data_root [32]byte) (*types.Transaction, error) {
	return _ValidatorRegistration.Contract.Deposit(&_ValidatorRegistration.TransactOpts, pubkey, withdrawal_credentials, signature, deposit_data_root)
}

// Deposit is a paid mutator transaction binding the contract method 0x22895118.
//
// Solidity: function deposit(bytes pubkey, bytes withdrawal_credentials, bytes signature, bytes32 deposit_data_root) returns()
func (_ValidatorRegistration *ValidatorRegistrationTransactorSession) Deposit(pubkey []byte, withdrawal_credentials []byte, signature []byte, deposit_data_root [32]byte) (*types.Transaction, error) {
	return _ValidatorRegistration.Contract.Deposit(&_ValidatorRegistration.TransactOpts, pubkey, withdrawal_credentials, signature, deposit_data_root)
}

// ValidatorRegistrationDepositEventIterator is returned from FilterDepositEvent and is used to iterate over the raw logs and unpacked data for DepositEvent events raised by the ValidatorRegistration contract.
type ValidatorRegistrationDepositEventIterator struct {
	Event *ValidatorRegistrationDepositEvent // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ValidatorRegistrationDepositEventIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ValidatorRegistrationDepositEvent)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ValidatorRegistrationDepositEvent)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ValidatorRegistrationDepositEventIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ValidatorRegistrationDepositEventIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ValidatorRegistrationDepositEvent represents a DepositEvent event raised by the ValidatorRegistration contract.
type ValidatorRegistrationDepositEvent struct {
	Pubkey                []byte
	WithdrawalCredentials []byte
	Amount                []byte
	Signature             []byte
	Index                 []byte
	Raw                   types.Log // Blockchain specific contextual infos
}

// FilterDepositEvent is a free log retrieval operation binding the contract event 0x649bbc62d0e31342afea4e5cd82d4049e7e1ee912fc0889aa790803be39038c5.
//
// Solidity: event DepositEvent(bytes pubkey, bytes withdrawal_credentials, bytes amount, bytes signature, bytes index)
func (_ValidatorRegistration *ValidatorRegistrationFilterer) FilterDepositEvent(opts *bind.FilterOpts) (*ValidatorRegistrationDepositEventIterator, error) {

	logs, sub, err := _ValidatorRegistration.contract.FilterLogs(opts, "DepositEvent")
	if err != nil {
		return nil, err
	}
	return &ValidatorRegistrationDepositEventIterator{contract: _ValidatorRegistration.contract, event: "DepositEvent", logs: logs, sub: sub}, nil
}

// WatchDepositEvent is a free log subscription operation binding the contract event 0x649bbc62d0e31342afea4e5cd82d4049e7e1ee912fc0889aa790803be39038c5.
//
// Solidity: event DepositEvent(bytes pubkey, bytes withdrawal_credentials, bytes amount, bytes signature, bytes index)
func (_ValidatorRegistration *ValidatorRegistrationFilterer) WatchDepositEvent(opts *bind.WatchOpts, sink chan<- *ValidatorRegistrationDepositEvent) (event.Subscription, error) {

	logs, sub, err := _ValidatorRegistration.contract.WatchLogs(opts, "DepositEvent")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ValidatorRegistrationDepositEvent)
				if err := _ValidatorRegistration.contract.UnpackLog(event, "DepositEvent", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseDepositEvent is a log parse operation binding the contract event 0x649bbc62d0e31342afea4e5cd82d4049e7e1ee912fc0889aa790803be39038c5.
//
// Solidity: event DepositEvent(bytes pubkey, bytes withdrawal_credentials, bytes amount, bytes signature, bytes index)
func (_ValidatorRegistration *ValidatorRegistrationFilterer) ParseDepositEvent(log types.Log) (*ValidatorRegistrationDepositEvent, error) {
	event := new(ValidatorRegistrationDepositEvent)
	if err := _ValidatorRegistration.contract.UnpackLog(event, "DepositEvent", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
