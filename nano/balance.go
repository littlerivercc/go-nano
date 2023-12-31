package nano

import (
	"encoding/binary"
	"errors"
	"math/big"

	"github.com/shopspring/decimal"
	"littleriver.cc/go-nano/nano/internal/uint128"
	"littleriver.cc/go-nano/nano/internal/util"
)

const (
	// BalanceSize represents the size of a balance in bytes.
	BalanceSize         = 16
	BalanceMaxPrecision = 33
)

type BalanceComp byte

const (
	BalanceCompEqual BalanceComp = iota
	BalanceCompBigger
	BalanceCompSmaller
)

var (
	units = map[string]decimal.Decimal{
		"raw":  decimal.New(1, 0),
		"uxrb": decimal.New(1, 18),
		"mxrb": decimal.New(1, 21),
		"xrb":  decimal.New(1, 24),
		"kxrb": decimal.New(1, 27),
		"Mxrb": decimal.New(1, 30),
		"Gxrb": decimal.New(1, 33),
	}

	ZeroBalance = Balance(uint128.Uint128{})

	ErrBadBalanceSize = errors.New("balances should be 16 bytes in size")
)

type Balance uint128.Uint128

// ParseBalance parses the given balance string.
func ParseBalance(s string, unit string) (Balance, error) {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return ZeroBalance, err
	}

	// zero is a special case
	if d.Equals(decimal.Zero) {
		return ZeroBalance, nil
	}

	d = d.Mul(units[unit])
	c := d.Coefficient()
	f := bigPow(10, int64(d.Exponent()))
	i := c.Mul(c, f)

	bytes := i.Bytes()
	balanceBytes := make([]byte, BalanceSize)
	copy(balanceBytes[len(balanceBytes)-len(bytes):], bytes)

	var balance Balance
	if err := balance.UnmarshalBinary(balanceBytes); err != nil {
		return ZeroBalance, err
	}

	return balance, nil
}

func ParseBalanceInts(hi uint64, lo uint64) Balance {
	return Balance(uint128.FromInts(hi, lo))
}

// Bytes returns the binary representation of this Balance with the given
// endianness.
func (b Balance) Bytes(order binary.ByteOrder) []byte {
	bytes := uint128.Uint128(b).GetBytes()

	switch order {
	case binary.BigEndian:
		return bytes
	case binary.LittleEndian:
		return util.ReverseBytes(bytes)
	default:
		panic("unsupported byte order")
	}
}

// Equal reports whether this balance and the given balance are equal.
func (b Balance) Equal(b2 Balance) bool {
	return uint128.Uint128(b).Equal(uint128.Uint128(b2))
}

func (b Balance) Add(n Balance) Balance {
	return Balance(uint128.Uint128(b).Add(uint128.Uint128(n)))
}

func (b Balance) Sub(n Balance) Balance {
	return Balance(uint128.Uint128(b).Sub(uint128.Uint128(n)))
}

func (b Balance) Compare(n Balance) BalanceComp {
	res := uint128.Uint128(b).Compare(uint128.Uint128(n))
	switch res {
	case 1:
		return BalanceCompBigger
	case -1:
		return BalanceCompSmaller
	case 0:
		return BalanceCompEqual
	default:
		panic("unexpected comparison result")
	}
}

// MarshalBinary implements the encoding.BinaryMarshaler interface.
func (b Balance) MarshalBinary() ([]byte, error) {
	return b.Bytes(binary.LittleEndian), nil
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface.
func (b *Balance) UnmarshalBinary(data []byte) error {
	if len(data) != BalanceSize {
		return ErrBadBalanceSize
	}

	*b = Balance(uint128.FromBytes(data))
	return nil
}

func (b Balance) BigInt() *big.Int {
	i := big.NewInt(0)
	i.SetBytes(b.Bytes(binary.BigEndian))
	return i
}

// UnitString returns a decimal representation of this uint128 converted to the
// given unit.
func (b Balance) UnitString(unit string, precision int32) string {
	d := decimal.NewFromBigInt(b.BigInt(), 0)
	return d.DivRound(units[unit], BalanceMaxPrecision).Truncate(precision).String()
}

// String implements the fmt.Stringer interface. It returns the balance in Mxrb
// with maximum precision.
func (b Balance) String() string {
	return b.UnitString("Mxrb", BalanceMaxPrecision)
}

func bigPow(base int64, exp int64) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(exp), nil)
}

// MarshalText implements the encoding.TextMarshaler interface.
func (b Balance) MarshalText() ([]byte, error) {
	return []byte(b.String()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (b *Balance) UnmarshalText(text []byte) error {
	balance, err := ParseBalance(string(text), "Mxrb")
	if err != nil {
		return err
	}

	*b = balance
	return nil
}
