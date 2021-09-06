package server

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"math/rand"
	"os"
	"strings"
	"syscall"

	cbn "github.com/celer-network/cBridge-go/cbridgenode"
	"github.com/celer-network/goutils/eth"
	"github.com/celer-network/goutils/log"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	eco "github.com/ethereum/go-ethereum/common"
	ethmath "github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/ssh/terminal"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	defaultTimeLockSafeMargin = 1200
)

func chkErr(e error, msg string) {
	if e != nil {
		fmt.Println("Err:", msg, e)
		os.Exit(1)
	}
}

// fatal on any error, guarantee return isn't nil
func ParseCfgFile(f string) (*cbn.CBridgeConfig, error) {
	raw, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, err
	}
	ret := new(cbn.CBridgeConfig)
	err = protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(raw, ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func getRandomPreImage() (Hash, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return Hash{}, err
	}
	return Bytes2Hash(key), nil
}

func getHashLockWithPreImage(preImage Hash) Hash {
	return Bytes2Hash(crypto.Keccak256(preImage.Bytes()))
}

type Hash = eco.Hash
type Addr = eco.Address

var (
	// set max_uint = 2**256-1
	// "0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	MaxUint256 = new(big.Int).SetBytes(Hex2Bytes(strings.Repeat("ff", 32)))
)

type AmtDelta struct {
	Id   uint32 // asset or st id
	Amt  *big.Int
	Plus bool // true means add, false means minus
}

// return is ALWAYS >= 0 ie. unsigned
func Bytes2Int(in []byte) *big.Int {
	return new(big.Int).SetBytes(in)
}

// given a big.Int, return like \x12\x34\x56, there may be better ways
// to be used for amount in pbtxt
func BigInt2PbStr(in *big.Int) string {
	hex := in.Text(16)
	if len(hex)%2 == 1 { // odd number, add 0
		hex = "0" + hex
	}
	ret := ""
	for i := 0; i <= len(hex)-2; i += 2 {
		ret += `\x` + hex[i:i+2]
	}
	return ret
}

func IsNegative(in *big.Int) bool {
	return in.Sign() == -1
}

// possible negative int256 to bytes in 2's complement if negative,
// returned []byte is always 32 bytes even it's all 0 when i==0
// if i >= 0, result is i.Bytes() with left padding to length 32
func SignInt256toBytes(i *big.Int) []byte {
	intcp := new(big.Int).Set(i) // have to use another bigInt b/c U256Bytes modifies input
	return ethmath.U256Bytes(intcp)
}

// bytes to possible negative Int256, if raw as unsigned int >= 2^255 (ie. 32 bytes and 1st bit is 1)
// it'll be treated as 2's complement for negative, eg. if raw = ff..ff (total 32) result will be -1.
func Bytes2SignInt256(raw []byte) *big.Int {
	orig := new(big.Int).SetBytes(raw)
	return ethmath.S256(orig)
}

// ========== Hex/Bytes ==========

// Hex2Bytes supports hex string with or without 0x prefix
// Calls hex.DecodeString directly and ignore err
// similar to ec.FromHex but better
func Hex2Bytes(s string) (b []byte) {
	if len(s) >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		s = s[2:]
	}
	// hex.DecodeString expects an even-length string
	if len(s)%2 == 1 {
		s = "0" + s
	}
	b, _ = hex.DecodeString(s)
	return b
}

// Bytes2Hex returns hex string without 0x prefix
func Bytes2Hex(b []byte) string {
	return hex.EncodeToString(b)
}

// ========== Address ==========

// Hex2Addr accepts hex string with or without 0x prefix and return Addr
func Hex2Addr(s string) Addr {
	return eco.BytesToAddress(Hex2Bytes(s))
}

// Addr2Hex returns hex without 0x
func Addr2Hex(a Addr) string {
	return Bytes2Hex(a[:])
}

// Bytes2Addr returns Address from b
// Addr.Bytes() does the reverse
func Bytes2Addr(b []byte) Addr {
	return eco.BytesToAddress(b)
}

// Hex2Addr accepts hex string with or without 0x prefix and return Addr
func Hex2Hash(s string) Hash {
	return eco.BytesToHash(Hex2Bytes(s))
}

func Bytes2Hash(b []byte) Hash {
	return eco.BytesToHash(b)
}

func GetTransactorConfig(ks, pwdDir string) (*eth.TransactorConfig, error) {
	ksjson, err := ioutil.ReadFile(ks)
	if err != nil {
		log.Fatalln("read ks json err:", err)
		return nil, err
	}
	ksPasswordStr, err := ReadPassword(ksjson, pwdDir)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return eth.NewTransactorConfig(string(ksjson), ksPasswordStr), nil
}

// Read a password from terminal or from a directory containing password files.
func ReadPassword(ksBytes []byte, pwdDir string) (string, error) {
	ksAddress, err := GetAddressFromKeystore(ksBytes)
	if err != nil {
		return "", err
	}

	if pwdDir != "" {
		pwdBytes, err2 := ioutil.ReadFile(pwdDir)
		if err2 != nil {
			return "", err2
		}
		return strings.Trim(string(pwdBytes), "\n"), nil
	}

	ksPasswordStr := ""
	if terminal.IsTerminal(syscall.Stdin) {
		fmt.Printf("Enter password for %s: ", ksAddress)
		ksPassword, err2 := terminal.ReadPassword(syscall.Stdin)
		if err2 != nil {
			return "", fmt.Errorf("Cannot read password from terminal: %w", err2)
		}
		ksPasswordStr = string(ksPassword)
	} else {
		reader := bufio.NewReader(os.Stdin)
		ksPwd, err2 := reader.ReadString('\n')
		if err2 != nil {
			return "", fmt.Errorf("Cannot read password from stdin: %w", err2)
		}
		ksPasswordStr = strings.TrimSuffix(ksPwd, "\n")
	}

	_, err = keystore.DecryptKey(ksBytes, ksPasswordStr)
	if err != nil {
		return "", err
	}
	return ksPasswordStr, nil
}

func GetAddressFromKeystore(ksBytes []byte) (string, error) {
	type ksStruct struct {
		Address string
	}
	var ks ksStruct
	if err := json.Unmarshal(ksBytes, &ks); err != nil {
		return "", err
	}
	return ks.Address, nil
}
