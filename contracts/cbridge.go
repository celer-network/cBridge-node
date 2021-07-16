// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contracts

import (
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
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// CBridgeABI is the input ABI used to generate the binding from.
const CBridgeABI = "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"transferId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"receiver\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"hashlock\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"timelock\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"srcChainId\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"srcTransferId\",\"type\":\"bytes32\"}],\"name\":\"LogNewTransferIn\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"transferId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"receiver\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"hashlock\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"timelock\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"dstChainId\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"dstAddress\",\"type\":\"address\"}],\"name\":\"LogNewTransferOut\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"transferId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"preimage\",\"type\":\"bytes32\"}],\"name\":\"LogTransferConfirmed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"transferId\",\"type\":\"bytes32\"}],\"name\":\"LogTransferRefunded\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_transferId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"_preimage\",\"type\":\"bytes32\"}],\"name\":\"confirm\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_transferId\",\"type\":\"bytes32\"}],\"name\":\"refund\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_dstAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"_hashlock\",\"type\":\"bytes32\"},{\"internalType\":\"uint64\",\"name\":\"_timelock\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"_srcChainId\",\"type\":\"uint64\"},{\"internalType\":\"bytes32\",\"name\":\"_srcTransferId\",\"type\":\"bytes32\"}],\"name\":\"transferIn\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_bridge\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"_hashlock\",\"type\":\"bytes32\"},{\"internalType\":\"uint64\",\"name\":\"_timelock\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"_dstChainId\",\"type\":\"uint64\"},{\"internalType\":\"address\",\"name\":\"_dstAddress\",\"type\":\"address\"}],\"name\":\"transferOut\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"transfers\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"receiver\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"hashlock\",\"type\":\"bytes32\"},{\"internalType\":\"uint64\",\"name\":\"timelock\",\"type\":\"uint64\"},{\"internalType\":\"enumCBridge.TransferStatus\",\"name\":\"status\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]"

// CBridgeBin is the compiled bytecode used for deploying new contracts.
var CBridgeBin = "0x608060405234801561001057600080fd5b50610ed8806100206000396000f3fe608060405234801561001057600080fd5b50600436106100575760003560e01c80633c64f04b1461005c57806346b40027146100df57806357f784ba146100f45780637249fbb614610107578063f63eba251461011a575b600080fd5b6100c361006a366004610d80565b6000602081905290815260409020805460018201546002830154600384015460048501546005909501546001600160a01b03948516959385169490921692909167ffffffffffffffff811690600160401b900460ff1687565b6040516100d69796959493929190610dd5565b60405180910390f35b6100f26100ed366004610c79565b61012d565b005b6100f2610102366004610d98565b6101ce565b6100f2610115366004610d80565b6103ee565b6100f2610128366004610cf0565b6105f6565b600061013c8888888888610688565b604080518281523360208201526001600160a01b03808c1692820192909252818a1660608201526080810189905260a0810188905267ffffffffffffffff80881660c0830152861660e08201529084166101008201529091507f2991ed59ec037d5602bdc81fb1f3fec360f9e68fd3c1a4efb49fb3f66a88e02190610120015b60405180910390a15050505050505050565b600082815260208181526040808320815160e08101835281546001600160a01b0390811682526001830154811694820194909452600282015490931691830191909152600380820154606084015260048201546080840152600582015467ffffffffffffffff811660a085015260c0840191600160401b90910460ff169081111561026957634e487b7160e01b600052602160045260246000fd5b600381111561028857634e487b7160e01b600052602160045260246000fd5b905250905060018160c0015160038111156102b357634e487b7160e01b600052602160045260246000fd5b146102fc5760405162461bcd60e51b81526020600482015260146024820152733737ba103832b73234b733903a3930b739b332b960611b60448201526064015b60405180910390fd5b604080516020810184905201604051602081830303815290604052805190602001208160800151146103655760405162461bcd60e51b8152602060048201526012602482015271696e636f727265637420707265696d61676560701b60448201526064016102f3565b60008381526020818152604091829020600501805460ff60401b1916680200000000000000001790558201516060830151918301516103b0926001600160a01b039091169190610953565b60408051848152602081018490527fb7ae890c7a4721f7ed769dabfeee74f0e0f5bcdaad9cab432ccea4d9fa435b50910160405180910390a1505050565b600081815260208181526040808320815160e08101835281546001600160a01b0390811682526001830154811694820194909452600282015490931691830191909152600380820154606084015260048201546080840152600582015467ffffffffffffffff811660a085015260c0840191600160401b90910460ff169081111561048957634e487b7160e01b600052602160045260246000fd5b60038111156104a857634e487b7160e01b600052602160045260246000fd5b905250905060018160c0015160038111156104d357634e487b7160e01b600052602160045260246000fd5b146105175760405162461bcd60e51b81526020600482015260146024820152733737ba103832b73234b733903a3930b739b332b960611b60448201526064016102f3565b428160a0015167ffffffffffffffff1611156105755760405162461bcd60e51b815260206004820152601760248201527f74696d656c6f636b206e6f74207965742070617373656400000000000000000060448201526064016102f3565b60008281526020819052604090819020600501805460ff60401b19166803000000000000000017905581516060830151918301516105bf926001600160a01b039091169190610953565b6040518281527f70a8f332cabb778f79acc5b97cbb4543970a2f1a34bd0773e4b3012931f752dc9060200160405180910390a15050565b60006106058888888888610688565b604080518281523360208201526001600160a01b03808c169282019290925290891660608201526080810188905260a0810187905267ffffffffffffffff80871660c0830152851660e082015261010081018490529091507f252a438c8e02dde6c26723283b3c95b1bf2550b882734770523f1d6767636f9c90610120016101bc565b60008084116106ca5760405162461bcd60e51b815260206004820152600e60248201526d1a5b9d985b1a5908185b5bdd5b9d60921b60448201526064016102f3565b428267ffffffffffffffff16116107165760405162461bcd60e51b815260206004820152601060248201526f696e76616c69642074696d656c6f636b60801b60448201526064016102f3565b6040516bffffffffffffffffffffffff1933606090811b8216602084015288901b1660348201526048810184905246606882015260880160408051601f198184030181529190528051602090910120905060008082815260208190526040902060050154600160401b900460ff1660038111156107a357634e487b7160e01b600052602160045260246000fd5b146107e25760405162461bcd60e51b815260206004820152600f60248201526e7472616e736665722065786973747360881b60448201526064016102f3565b6107f76001600160a01b0386163330876109bb565b6040518060e00160405280336001600160a01b03168152602001876001600160a01b03168152602001866001600160a01b031681526020018581526020018481526020018367ffffffffffffffff1681526020016001600381111561086c57634e487b7160e01b600052602160045260246000fd5b905260008281526020818152604091829020835181546001600160a01b039182166001600160a01b031991821617835592850151600183018054918316918516919091179055928401516002820180549190941692169190911790915560608201516003808301919091556080830151600483015560a083015160058301805467ffffffffffffffff90921667ffffffffffffffff1983168117825560c086015193919268ffffffffffffffffff19161790600160401b90849081111561094357634e487b7160e01b600052602160045260246000fd5b0217905550505095945050505050565b6040516001600160a01b0383166024820152604481018290526109b690849063a9059cbb60e01b906064015b60408051601f198184030181529190526020810180516001600160e01b03166001600160e01b0319909316929092179091526109f9565b505050565b6040516001600160a01b03808516602483015283166044820152606481018290526109f39085906323b872dd60e01b9060840161097f565b50505050565b6000610a4e826040518060400160405280602081526020017f5361666545524332303a206c6f772d6c6576656c2063616c6c206661696c6564815250856001600160a01b0316610acb9092919063ffffffff16565b8051909150156109b65780806020019051810190610a6c9190610d60565b6109b65760405162461bcd60e51b815260206004820152602a60248201527f5361666545524332303a204552433230206f7065726174696f6e20646964206e6044820152691bdd081cdd58d8d9595960b21b60648201526084016102f3565b6060610ada8484600085610ae4565b90505b9392505050565b606082471015610b455760405162461bcd60e51b815260206004820152602660248201527f416464726573733a20696e73756666696369656e742062616c616e636520666f6044820152651c8818d85b1b60d21b60648201526084016102f3565b843b610b935760405162461bcd60e51b815260206004820152601d60248201527f416464726573733a2063616c6c20746f206e6f6e2d636f6e747261637400000060448201526064016102f3565b600080866001600160a01b03168587604051610baf9190610db9565b60006040518083038185875af1925050503d8060008114610bec576040519150601f19603f3d011682016040523d82523d6000602084013e610bf1565b606091505b5091509150610c01828286610c0c565b979650505050505050565b60608315610c1b575081610add565b825115610c2b5782518084602001fd5b8160405162461bcd60e51b81526004016102f39190610e43565b80356001600160a01b0381168114610c5c57600080fd5b919050565b803567ffffffffffffffff81168114610c5c57600080fd5b600080600080600080600060e0888a031215610c93578283fd5b610c9c88610c45565b9650610caa60208901610c45565b95506040880135945060608801359350610cc660808901610c61565b9250610cd460a08901610c61565b9150610ce260c08901610c45565b905092959891949750929550565b600080600080600080600060e0888a031215610d0a578283fd5b610d1388610c45565b9650610d2160208901610c45565b95506040880135945060608801359350610d3d60808901610c61565b9250610d4b60a08901610c61565b915060c0880135905092959891949750929550565b600060208284031215610d71578081fd5b81518015158114610add578182fd5b600060208284031215610d91578081fd5b5035919050565b60008060408385031215610daa578182fd5b50508035926020909101359150565b60008251610dcb818460208701610e76565b9190910192915050565b6001600160a01b038881168252878116602083015286166040820152606081018590526080810184905267ffffffffffffffff831660a082015260e0810160048310610e3157634e487b7160e01b600052602160045260246000fd5b8260c083015298975050505050505050565b6020815260008251806020840152610e62816040850160208701610e76565b601f01601f19169190910160400192915050565b60005b83811015610e91578181015183820152602001610e79565b838111156109f3575050600091015256fea2646970667358221220b4b723539d9a54ae4db5ceb90d93702b2fa0a6deafeaa58b2547b41c15c73c7064736f6c63430008040033"

// DeployCBridge deploys a new Ethereum contract, binding an instance of CBridge to it.
func DeployCBridge(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *CBridge, error) {
	parsed, err := abi.JSON(strings.NewReader(CBridgeABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(CBridgeBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &CBridge{CBridgeCaller: CBridgeCaller{contract: contract}, CBridgeTransactor: CBridgeTransactor{contract: contract}, CBridgeFilterer: CBridgeFilterer{contract: contract}}, nil
}

// CBridge is an auto generated Go binding around an Ethereum contract.
type CBridge struct {
	CBridgeCaller     // Read-only binding to the contract
	CBridgeTransactor // Write-only binding to the contract
	CBridgeFilterer   // Log filterer for contract events
}

// CBridgeCaller is an auto generated read-only Go binding around an Ethereum contract.
type CBridgeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CBridgeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type CBridgeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CBridgeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type CBridgeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CBridgeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type CBridgeSession struct {
	Contract     *CBridge          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// CBridgeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type CBridgeCallerSession struct {
	Contract *CBridgeCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// CBridgeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type CBridgeTransactorSession struct {
	Contract     *CBridgeTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// CBridgeRaw is an auto generated low-level Go binding around an Ethereum contract.
type CBridgeRaw struct {
	Contract *CBridge // Generic contract binding to access the raw methods on
}

// CBridgeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type CBridgeCallerRaw struct {
	Contract *CBridgeCaller // Generic read-only contract binding to access the raw methods on
}

// CBridgeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type CBridgeTransactorRaw struct {
	Contract *CBridgeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewCBridge creates a new instance of CBridge, bound to a specific deployed contract.
func NewCBridge(address common.Address, backend bind.ContractBackend) (*CBridge, error) {
	contract, err := bindCBridge(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &CBridge{CBridgeCaller: CBridgeCaller{contract: contract}, CBridgeTransactor: CBridgeTransactor{contract: contract}, CBridgeFilterer: CBridgeFilterer{contract: contract}}, nil
}

// NewCBridgeCaller creates a new read-only instance of CBridge, bound to a specific deployed contract.
func NewCBridgeCaller(address common.Address, caller bind.ContractCaller) (*CBridgeCaller, error) {
	contract, err := bindCBridge(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &CBridgeCaller{contract: contract}, nil
}

// NewCBridgeTransactor creates a new write-only instance of CBridge, bound to a specific deployed contract.
func NewCBridgeTransactor(address common.Address, transactor bind.ContractTransactor) (*CBridgeTransactor, error) {
	contract, err := bindCBridge(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &CBridgeTransactor{contract: contract}, nil
}

// NewCBridgeFilterer creates a new log filterer instance of CBridge, bound to a specific deployed contract.
func NewCBridgeFilterer(address common.Address, filterer bind.ContractFilterer) (*CBridgeFilterer, error) {
	contract, err := bindCBridge(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &CBridgeFilterer{contract: contract}, nil
}

// bindCBridge binds a generic wrapper to an already deployed contract.
func bindCBridge(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(CBridgeABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CBridge *CBridgeRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CBridge.Contract.CBridgeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CBridge *CBridgeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CBridge.Contract.CBridgeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CBridge *CBridgeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CBridge.Contract.CBridgeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CBridge *CBridgeCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CBridge.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CBridge *CBridgeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CBridge.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CBridge *CBridgeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CBridge.Contract.contract.Transact(opts, method, params...)
}

// Transfers is a free data retrieval call binding the contract method 0x3c64f04b.
//
// Solidity: function transfers(bytes32 ) view returns(address sender, address receiver, address token, uint256 amount, bytes32 hashlock, uint64 timelock, uint8 status)
func (_CBridge *CBridgeCaller) Transfers(opts *bind.CallOpts, arg0 [32]byte) (struct {
	Sender   common.Address
	Receiver common.Address
	Token    common.Address
	Amount   *big.Int
	Hashlock [32]byte
	Timelock uint64
	Status   uint8
}, error) {
	var out []interface{}
	err := _CBridge.contract.Call(opts, &out, "transfers", arg0)

	outstruct := new(struct {
		Sender   common.Address
		Receiver common.Address
		Token    common.Address
		Amount   *big.Int
		Hashlock [32]byte
		Timelock uint64
		Status   uint8
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Sender = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.Receiver = *abi.ConvertType(out[1], new(common.Address)).(*common.Address)
	outstruct.Token = *abi.ConvertType(out[2], new(common.Address)).(*common.Address)
	outstruct.Amount = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.Hashlock = *abi.ConvertType(out[4], new([32]byte)).(*[32]byte)
	outstruct.Timelock = *abi.ConvertType(out[5], new(uint64)).(*uint64)
	outstruct.Status = *abi.ConvertType(out[6], new(uint8)).(*uint8)

	return *outstruct, err

}

// Transfers is a free data retrieval call binding the contract method 0x3c64f04b.
//
// Solidity: function transfers(bytes32 ) view returns(address sender, address receiver, address token, uint256 amount, bytes32 hashlock, uint64 timelock, uint8 status)
func (_CBridge *CBridgeSession) Transfers(arg0 [32]byte) (struct {
	Sender   common.Address
	Receiver common.Address
	Token    common.Address
	Amount   *big.Int
	Hashlock [32]byte
	Timelock uint64
	Status   uint8
}, error) {
	return _CBridge.Contract.Transfers(&_CBridge.CallOpts, arg0)
}

// Transfers is a free data retrieval call binding the contract method 0x3c64f04b.
//
// Solidity: function transfers(bytes32 ) view returns(address sender, address receiver, address token, uint256 amount, bytes32 hashlock, uint64 timelock, uint8 status)
func (_CBridge *CBridgeCallerSession) Transfers(arg0 [32]byte) (struct {
	Sender   common.Address
	Receiver common.Address
	Token    common.Address
	Amount   *big.Int
	Hashlock [32]byte
	Timelock uint64
	Status   uint8
}, error) {
	return _CBridge.Contract.Transfers(&_CBridge.CallOpts, arg0)
}

// Confirm is a paid mutator transaction binding the contract method 0x57f784ba.
//
// Solidity: function confirm(bytes32 _transferId, bytes32 _preimage) returns()
func (_CBridge *CBridgeTransactor) Confirm(opts *bind.TransactOpts, _transferId [32]byte, _preimage [32]byte) (*types.Transaction, error) {
	return _CBridge.contract.Transact(opts, "confirm", _transferId, _preimage)
}

// Confirm is a paid mutator transaction binding the contract method 0x57f784ba.
//
// Solidity: function confirm(bytes32 _transferId, bytes32 _preimage) returns()
func (_CBridge *CBridgeSession) Confirm(_transferId [32]byte, _preimage [32]byte) (*types.Transaction, error) {
	return _CBridge.Contract.Confirm(&_CBridge.TransactOpts, _transferId, _preimage)
}

// Confirm is a paid mutator transaction binding the contract method 0x57f784ba.
//
// Solidity: function confirm(bytes32 _transferId, bytes32 _preimage) returns()
func (_CBridge *CBridgeTransactorSession) Confirm(_transferId [32]byte, _preimage [32]byte) (*types.Transaction, error) {
	return _CBridge.Contract.Confirm(&_CBridge.TransactOpts, _transferId, _preimage)
}

// Refund is a paid mutator transaction binding the contract method 0x7249fbb6.
//
// Solidity: function refund(bytes32 _transferId) returns()
func (_CBridge *CBridgeTransactor) Refund(opts *bind.TransactOpts, _transferId [32]byte) (*types.Transaction, error) {
	return _CBridge.contract.Transact(opts, "refund", _transferId)
}

// Refund is a paid mutator transaction binding the contract method 0x7249fbb6.
//
// Solidity: function refund(bytes32 _transferId) returns()
func (_CBridge *CBridgeSession) Refund(_transferId [32]byte) (*types.Transaction, error) {
	return _CBridge.Contract.Refund(&_CBridge.TransactOpts, _transferId)
}

// Refund is a paid mutator transaction binding the contract method 0x7249fbb6.
//
// Solidity: function refund(bytes32 _transferId) returns()
func (_CBridge *CBridgeTransactorSession) Refund(_transferId [32]byte) (*types.Transaction, error) {
	return _CBridge.Contract.Refund(&_CBridge.TransactOpts, _transferId)
}

// TransferIn is a paid mutator transaction binding the contract method 0xf63eba25.
//
// Solidity: function transferIn(address _dstAddress, address _token, uint256 _amount, bytes32 _hashlock, uint64 _timelock, uint64 _srcChainId, bytes32 _srcTransferId) returns()
func (_CBridge *CBridgeTransactor) TransferIn(opts *bind.TransactOpts, _dstAddress common.Address, _token common.Address, _amount *big.Int, _hashlock [32]byte, _timelock uint64, _srcChainId uint64, _srcTransferId [32]byte) (*types.Transaction, error) {
	return _CBridge.contract.Transact(opts, "transferIn", _dstAddress, _token, _amount, _hashlock, _timelock, _srcChainId, _srcTransferId)
}

// TransferIn is a paid mutator transaction binding the contract method 0xf63eba25.
//
// Solidity: function transferIn(address _dstAddress, address _token, uint256 _amount, bytes32 _hashlock, uint64 _timelock, uint64 _srcChainId, bytes32 _srcTransferId) returns()
func (_CBridge *CBridgeSession) TransferIn(_dstAddress common.Address, _token common.Address, _amount *big.Int, _hashlock [32]byte, _timelock uint64, _srcChainId uint64, _srcTransferId [32]byte) (*types.Transaction, error) {
	return _CBridge.Contract.TransferIn(&_CBridge.TransactOpts, _dstAddress, _token, _amount, _hashlock, _timelock, _srcChainId, _srcTransferId)
}

// TransferIn is a paid mutator transaction binding the contract method 0xf63eba25.
//
// Solidity: function transferIn(address _dstAddress, address _token, uint256 _amount, bytes32 _hashlock, uint64 _timelock, uint64 _srcChainId, bytes32 _srcTransferId) returns()
func (_CBridge *CBridgeTransactorSession) TransferIn(_dstAddress common.Address, _token common.Address, _amount *big.Int, _hashlock [32]byte, _timelock uint64, _srcChainId uint64, _srcTransferId [32]byte) (*types.Transaction, error) {
	return _CBridge.Contract.TransferIn(&_CBridge.TransactOpts, _dstAddress, _token, _amount, _hashlock, _timelock, _srcChainId, _srcTransferId)
}

// TransferOut is a paid mutator transaction binding the contract method 0x46b40027.
//
// Solidity: function transferOut(address _bridge, address _token, uint256 _amount, bytes32 _hashlock, uint64 _timelock, uint64 _dstChainId, address _dstAddress) returns()
func (_CBridge *CBridgeTransactor) TransferOut(opts *bind.TransactOpts, _bridge common.Address, _token common.Address, _amount *big.Int, _hashlock [32]byte, _timelock uint64, _dstChainId uint64, _dstAddress common.Address) (*types.Transaction, error) {
	return _CBridge.contract.Transact(opts, "transferOut", _bridge, _token, _amount, _hashlock, _timelock, _dstChainId, _dstAddress)
}

// TransferOut is a paid mutator transaction binding the contract method 0x46b40027.
//
// Solidity: function transferOut(address _bridge, address _token, uint256 _amount, bytes32 _hashlock, uint64 _timelock, uint64 _dstChainId, address _dstAddress) returns()
func (_CBridge *CBridgeSession) TransferOut(_bridge common.Address, _token common.Address, _amount *big.Int, _hashlock [32]byte, _timelock uint64, _dstChainId uint64, _dstAddress common.Address) (*types.Transaction, error) {
	return _CBridge.Contract.TransferOut(&_CBridge.TransactOpts, _bridge, _token, _amount, _hashlock, _timelock, _dstChainId, _dstAddress)
}

// TransferOut is a paid mutator transaction binding the contract method 0x46b40027.
//
// Solidity: function transferOut(address _bridge, address _token, uint256 _amount, bytes32 _hashlock, uint64 _timelock, uint64 _dstChainId, address _dstAddress) returns()
func (_CBridge *CBridgeTransactorSession) TransferOut(_bridge common.Address, _token common.Address, _amount *big.Int, _hashlock [32]byte, _timelock uint64, _dstChainId uint64, _dstAddress common.Address) (*types.Transaction, error) {
	return _CBridge.Contract.TransferOut(&_CBridge.TransactOpts, _bridge, _token, _amount, _hashlock, _timelock, _dstChainId, _dstAddress)
}

// CBridgeLogNewTransferInIterator is returned from FilterLogNewTransferIn and is used to iterate over the raw logs and unpacked data for LogNewTransferIn events raised by the CBridge contract.
type CBridgeLogNewTransferInIterator struct {
	Event *CBridgeLogNewTransferIn // Event containing the contract specifics and raw log

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
func (it *CBridgeLogNewTransferInIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CBridgeLogNewTransferIn)
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
		it.Event = new(CBridgeLogNewTransferIn)
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
func (it *CBridgeLogNewTransferInIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CBridgeLogNewTransferInIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CBridgeLogNewTransferIn represents a LogNewTransferIn event raised by the CBridge contract.
type CBridgeLogNewTransferIn struct {
	TransferId    [32]byte
	Sender        common.Address
	Receiver      common.Address
	Token         common.Address
	Amount        *big.Int
	Hashlock      [32]byte
	Timelock      uint64
	SrcChainId    uint64
	SrcTransferId [32]byte
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterLogNewTransferIn is a free log retrieval operation binding the contract event 0x252a438c8e02dde6c26723283b3c95b1bf2550b882734770523f1d6767636f9c.
//
// Solidity: event LogNewTransferIn(bytes32 transferId, address sender, address receiver, address token, uint256 amount, bytes32 hashlock, uint64 timelock, uint64 srcChainId, bytes32 srcTransferId)
func (_CBridge *CBridgeFilterer) FilterLogNewTransferIn(opts *bind.FilterOpts) (*CBridgeLogNewTransferInIterator, error) {

	logs, sub, err := _CBridge.contract.FilterLogs(opts, "LogNewTransferIn")
	if err != nil {
		return nil, err
	}
	return &CBridgeLogNewTransferInIterator{contract: _CBridge.contract, event: "LogNewTransferIn", logs: logs, sub: sub}, nil
}

// WatchLogNewTransferIn is a free log subscription operation binding the contract event 0x252a438c8e02dde6c26723283b3c95b1bf2550b882734770523f1d6767636f9c.
//
// Solidity: event LogNewTransferIn(bytes32 transferId, address sender, address receiver, address token, uint256 amount, bytes32 hashlock, uint64 timelock, uint64 srcChainId, bytes32 srcTransferId)
func (_CBridge *CBridgeFilterer) WatchLogNewTransferIn(opts *bind.WatchOpts, sink chan<- *CBridgeLogNewTransferIn) (event.Subscription, error) {

	logs, sub, err := _CBridge.contract.WatchLogs(opts, "LogNewTransferIn")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CBridgeLogNewTransferIn)
				if err := _CBridge.contract.UnpackLog(event, "LogNewTransferIn", log); err != nil {
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

// ParseLogNewTransferIn is a log parse operation binding the contract event 0x252a438c8e02dde6c26723283b3c95b1bf2550b882734770523f1d6767636f9c.
//
// Solidity: event LogNewTransferIn(bytes32 transferId, address sender, address receiver, address token, uint256 amount, bytes32 hashlock, uint64 timelock, uint64 srcChainId, bytes32 srcTransferId)
func (_CBridge *CBridgeFilterer) ParseLogNewTransferIn(log types.Log) (*CBridgeLogNewTransferIn, error) {
	event := new(CBridgeLogNewTransferIn)
	if err := _CBridge.contract.UnpackLog(event, "LogNewTransferIn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CBridgeLogNewTransferOutIterator is returned from FilterLogNewTransferOut and is used to iterate over the raw logs and unpacked data for LogNewTransferOut events raised by the CBridge contract.
type CBridgeLogNewTransferOutIterator struct {
	Event *CBridgeLogNewTransferOut // Event containing the contract specifics and raw log

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
func (it *CBridgeLogNewTransferOutIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CBridgeLogNewTransferOut)
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
		it.Event = new(CBridgeLogNewTransferOut)
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
func (it *CBridgeLogNewTransferOutIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CBridgeLogNewTransferOutIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CBridgeLogNewTransferOut represents a LogNewTransferOut event raised by the CBridge contract.
type CBridgeLogNewTransferOut struct {
	TransferId [32]byte
	Sender     common.Address
	Receiver   common.Address
	Token      common.Address
	Amount     *big.Int
	Hashlock   [32]byte
	Timelock   uint64
	DstChainId uint64
	DstAddress common.Address
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterLogNewTransferOut is a free log retrieval operation binding the contract event 0x2991ed59ec037d5602bdc81fb1f3fec360f9e68fd3c1a4efb49fb3f66a88e021.
//
// Solidity: event LogNewTransferOut(bytes32 transferId, address sender, address receiver, address token, uint256 amount, bytes32 hashlock, uint64 timelock, uint64 dstChainId, address dstAddress)
func (_CBridge *CBridgeFilterer) FilterLogNewTransferOut(opts *bind.FilterOpts) (*CBridgeLogNewTransferOutIterator, error) {

	logs, sub, err := _CBridge.contract.FilterLogs(opts, "LogNewTransferOut")
	if err != nil {
		return nil, err
	}
	return &CBridgeLogNewTransferOutIterator{contract: _CBridge.contract, event: "LogNewTransferOut", logs: logs, sub: sub}, nil
}

// WatchLogNewTransferOut is a free log subscription operation binding the contract event 0x2991ed59ec037d5602bdc81fb1f3fec360f9e68fd3c1a4efb49fb3f66a88e021.
//
// Solidity: event LogNewTransferOut(bytes32 transferId, address sender, address receiver, address token, uint256 amount, bytes32 hashlock, uint64 timelock, uint64 dstChainId, address dstAddress)
func (_CBridge *CBridgeFilterer) WatchLogNewTransferOut(opts *bind.WatchOpts, sink chan<- *CBridgeLogNewTransferOut) (event.Subscription, error) {

	logs, sub, err := _CBridge.contract.WatchLogs(opts, "LogNewTransferOut")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CBridgeLogNewTransferOut)
				if err := _CBridge.contract.UnpackLog(event, "LogNewTransferOut", log); err != nil {
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

// ParseLogNewTransferOut is a log parse operation binding the contract event 0x2991ed59ec037d5602bdc81fb1f3fec360f9e68fd3c1a4efb49fb3f66a88e021.
//
// Solidity: event LogNewTransferOut(bytes32 transferId, address sender, address receiver, address token, uint256 amount, bytes32 hashlock, uint64 timelock, uint64 dstChainId, address dstAddress)
func (_CBridge *CBridgeFilterer) ParseLogNewTransferOut(log types.Log) (*CBridgeLogNewTransferOut, error) {
	event := new(CBridgeLogNewTransferOut)
	if err := _CBridge.contract.UnpackLog(event, "LogNewTransferOut", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CBridgeLogTransferConfirmedIterator is returned from FilterLogTransferConfirmed and is used to iterate over the raw logs and unpacked data for LogTransferConfirmed events raised by the CBridge contract.
type CBridgeLogTransferConfirmedIterator struct {
	Event *CBridgeLogTransferConfirmed // Event containing the contract specifics and raw log

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
func (it *CBridgeLogTransferConfirmedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CBridgeLogTransferConfirmed)
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
		it.Event = new(CBridgeLogTransferConfirmed)
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
func (it *CBridgeLogTransferConfirmedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CBridgeLogTransferConfirmedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CBridgeLogTransferConfirmed represents a LogTransferConfirmed event raised by the CBridge contract.
type CBridgeLogTransferConfirmed struct {
	TransferId [32]byte
	Preimage   [32]byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterLogTransferConfirmed is a free log retrieval operation binding the contract event 0xb7ae890c7a4721f7ed769dabfeee74f0e0f5bcdaad9cab432ccea4d9fa435b50.
//
// Solidity: event LogTransferConfirmed(bytes32 transferId, bytes32 preimage)
func (_CBridge *CBridgeFilterer) FilterLogTransferConfirmed(opts *bind.FilterOpts) (*CBridgeLogTransferConfirmedIterator, error) {

	logs, sub, err := _CBridge.contract.FilterLogs(opts, "LogTransferConfirmed")
	if err != nil {
		return nil, err
	}
	return &CBridgeLogTransferConfirmedIterator{contract: _CBridge.contract, event: "LogTransferConfirmed", logs: logs, sub: sub}, nil
}

// WatchLogTransferConfirmed is a free log subscription operation binding the contract event 0xb7ae890c7a4721f7ed769dabfeee74f0e0f5bcdaad9cab432ccea4d9fa435b50.
//
// Solidity: event LogTransferConfirmed(bytes32 transferId, bytes32 preimage)
func (_CBridge *CBridgeFilterer) WatchLogTransferConfirmed(opts *bind.WatchOpts, sink chan<- *CBridgeLogTransferConfirmed) (event.Subscription, error) {

	logs, sub, err := _CBridge.contract.WatchLogs(opts, "LogTransferConfirmed")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CBridgeLogTransferConfirmed)
				if err := _CBridge.contract.UnpackLog(event, "LogTransferConfirmed", log); err != nil {
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

// ParseLogTransferConfirmed is a log parse operation binding the contract event 0xb7ae890c7a4721f7ed769dabfeee74f0e0f5bcdaad9cab432ccea4d9fa435b50.
//
// Solidity: event LogTransferConfirmed(bytes32 transferId, bytes32 preimage)
func (_CBridge *CBridgeFilterer) ParseLogTransferConfirmed(log types.Log) (*CBridgeLogTransferConfirmed, error) {
	event := new(CBridgeLogTransferConfirmed)
	if err := _CBridge.contract.UnpackLog(event, "LogTransferConfirmed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CBridgeLogTransferRefundedIterator is returned from FilterLogTransferRefunded and is used to iterate over the raw logs and unpacked data for LogTransferRefunded events raised by the CBridge contract.
type CBridgeLogTransferRefundedIterator struct {
	Event *CBridgeLogTransferRefunded // Event containing the contract specifics and raw log

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
func (it *CBridgeLogTransferRefundedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CBridgeLogTransferRefunded)
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
		it.Event = new(CBridgeLogTransferRefunded)
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
func (it *CBridgeLogTransferRefundedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CBridgeLogTransferRefundedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CBridgeLogTransferRefunded represents a LogTransferRefunded event raised by the CBridge contract.
type CBridgeLogTransferRefunded struct {
	TransferId [32]byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterLogTransferRefunded is a free log retrieval operation binding the contract event 0x70a8f332cabb778f79acc5b97cbb4543970a2f1a34bd0773e4b3012931f752dc.
//
// Solidity: event LogTransferRefunded(bytes32 transferId)
func (_CBridge *CBridgeFilterer) FilterLogTransferRefunded(opts *bind.FilterOpts) (*CBridgeLogTransferRefundedIterator, error) {

	logs, sub, err := _CBridge.contract.FilterLogs(opts, "LogTransferRefunded")
	if err != nil {
		return nil, err
	}
	return &CBridgeLogTransferRefundedIterator{contract: _CBridge.contract, event: "LogTransferRefunded", logs: logs, sub: sub}, nil
}

// WatchLogTransferRefunded is a free log subscription operation binding the contract event 0x70a8f332cabb778f79acc5b97cbb4543970a2f1a34bd0773e4b3012931f752dc.
//
// Solidity: event LogTransferRefunded(bytes32 transferId)
func (_CBridge *CBridgeFilterer) WatchLogTransferRefunded(opts *bind.WatchOpts, sink chan<- *CBridgeLogTransferRefunded) (event.Subscription, error) {

	logs, sub, err := _CBridge.contract.WatchLogs(opts, "LogTransferRefunded")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CBridgeLogTransferRefunded)
				if err := _CBridge.contract.UnpackLog(event, "LogTransferRefunded", log); err != nil {
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

// ParseLogTransferRefunded is a log parse operation binding the contract event 0x70a8f332cabb778f79acc5b97cbb4543970a2f1a34bd0773e4b3012931f752dc.
//
// Solidity: event LogTransferRefunded(bytes32 transferId)
func (_CBridge *CBridgeFilterer) ParseLogTransferRefunded(log types.Log) (*CBridgeLogTransferRefunded, error) {
	event := new(CBridgeLogTransferRefunded)
	if err := _CBridge.contract.UnpackLog(event, "LogTransferRefunded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
