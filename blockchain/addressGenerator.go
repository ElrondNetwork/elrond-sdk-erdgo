package blockchain

import (
	"bytes"

	elrondCore "github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data/typeConverters/uint64ByteSlice"
	"github.com/ElrondNetwork/elrond-go-core/hashing"
	"github.com/ElrondNetwork/elrond-go-core/hashing/keccak"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/ElrondNetwork/elrond-go/process/smartContract/hooks"
	"github.com/ElrondNetwork/elrond-sdk-erdgo/core"
	"github.com/ElrondNetwork/elrond-sdk-erdgo/data"
	"github.com/ElrondNetwork/elrond-sdk-erdgo/disabled"
	"github.com/ElrondNetwork/elrond-sdk-erdgo/storage"
)

const accountStartNonce = uint64(0)

var initialDNSAddress = bytes.Repeat([]byte{1}, 32)

// addressGenerator is used to generate some addresses based on elrond-go logic
type addressGenerator struct {
	coordinator    *shardCoordinator
	blockChainHook process.BlockChainHookHandler
	hasher         hashing.Hasher
}

// NewAddressGenerator will create an address generator instance
func NewAddressGenerator(coordinator *shardCoordinator) (*addressGenerator, error) {
	if check.IfNil(coordinator) {
		return nil, ErrNilShardCoordinator
	}

	builtInFuncs := &disabled.BuiltInFunctionContainer{}
	var argsHook = hooks.ArgBlockChainHook{
		Accounts:           &disabled.Accounts{},
		PubkeyConv:         core.AddressPublicKeyConverter,
		StorageService:     &disabled.StorageService{},
		BlockChain:         &disabled.Blockchain{},
		ShardCoordinator:   &disabled.ElrondShardCoordinator{},
		Marshalizer:        &marshal.JsonMarshalizer{},
		Uint64Converter:    uint64ByteSlice.NewBigEndianConverter(),
		BuiltInFunctions:   builtInFuncs,
		DataPool:           &disabled.DataPool{},
		CompiledSCPool:     storage.NewMapCacher(),
		NilCompiledSCStore: true,
		NFTStorageHandler:  &disabled.SimpleESDTNFTStorageHandler{},
		EpochNotifier:      &disabled.EpochNotifier{},
	}
	blockchainHook, err := hooks.NewBlockChainHookImpl(argsHook)
	if err != nil {
		return nil, err
	}

	return &addressGenerator{
		coordinator:    coordinator,
		blockChainHook: blockchainHook,
		hasher:         keccak.NewKeccak(),
	}, nil
}

// CompatibleDNSAddress will return the compatible DNS address providing the shard ID
func (ag *addressGenerator) CompatibleDNSAddress(shardId byte) (core.AddressHandler, error) {
	addressLen := len(initialDNSAddress)
	shardInBytes := []byte{0, shardId}

	newDNSPk := string(initialDNSAddress[:(addressLen-elrondCore.ShardIdentiferLen)]) + string(shardInBytes)
	newDNSAddress, err := ag.blockChainHook.NewAddress([]byte(newDNSPk), accountStartNonce, factory.ArwenVirtualMachine)
	if err != nil {
		return nil, err
	}

	return data.NewAddressFromBytes(newDNSAddress), err
}

// CompatibleDNSAddressFromUsername will return the compatible DNS address providing the username
func (ag *addressGenerator) CompatibleDNSAddressFromUsername(username string) (core.AddressHandler, error) {
	hash := ag.hasher.Compute(username)
	lastByte := hash[len(hash)-1]
	return ag.CompatibleDNSAddress(lastByte)
}

// ComputeArwenScAddress will return the smart contract address that will be generated by the Arwen VM providing
// the owner's address & nonce
func (ag *addressGenerator) ComputeArwenScAddress(address core.AddressHandler, nonce uint64) (core.AddressHandler, error) {
	if check.IfNil(address) {
		return nil, ErrNilAddress
	}

	scAddressBytes, err := ag.blockChainHook.NewAddress(address.AddressBytes(), nonce, factory.ArwenVirtualMachine)
	if err != nil {
		return nil, err
	}

	return data.NewAddressFromBytes(scAddressBytes), nil
}
