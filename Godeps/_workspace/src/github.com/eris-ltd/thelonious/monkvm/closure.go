package monkvm

// TODO Re write VM to use values instead of big integers?

import (
	"math/big"

	"github.com/eris-ltd/modules/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkstate"
	"github.com/eris-ltd/modules/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
)

type ClosureRef interface {
	ReturnGas(*big.Int, *big.Int)
	Address() []byte
	Object() *monkstate.StateObject
	GetStorage(*big.Int) *monkutil.Value
	SetStorage(*big.Int, *monkutil.Value)
}

// Basic inline closure object which implement the 'closure' interface
type Closure struct {
	caller  ClosureRef
	object  *monkstate.StateObject
	Code    []byte
	message *monkstate.Message

	Gas, UsedGas, Price *big.Int

	Args []byte
}

// Create a new closure for the given data items
func NewClosure(msg *monkstate.Message, caller ClosureRef, object *monkstate.StateObject, code []byte, gas, price *big.Int) *Closure {
	c := &Closure{message: msg, caller: caller, object: object, Code: code, Args: nil}

	// Gas should be a pointer so it can safely be reduced through the run
	// This pointer will be off the state transition
	c.Gas = gas //new(big.Int).Set(gas)
	// In most cases price and value are pointers to transaction objects
	// and we don't want the transaction's values to change.
	c.Price = new(big.Int).Set(price)
	c.UsedGas = new(big.Int)

	return c
}

// Retuns the x element in data slice
func (c *Closure) GetStorage(x *big.Int) *monkutil.Value {
	m := c.object.GetStorage(x)
	if m == nil {
		return monkutil.EmptyValue()
	}

	return m
}

func (c *Closure) Get(x *big.Int) *monkutil.Value {
	return c.Gets(x, big.NewInt(1))
}

func (c *Closure) Gets(x, y *big.Int) *monkutil.Value {
	if x.Int64() >= int64(len(c.Code)) || y.Int64() >= int64(len(c.Code)) {
		return monkutil.NewValue(0)
	}

	partial := c.Code[x.Int64() : x.Int64()+y.Int64()]

	return monkutil.NewValue(partial)
}

func (c *Closure) SetStorage(x *big.Int, val *monkutil.Value) {
	c.object.SetStorage(x, val)
}

func (c *Closure) Address() []byte {
	return c.object.Address()
}

func (c *Closure) Call(vm *Vm, args []byte) ([]byte, *big.Int, error) {
	c.Args = args

	ret, err := vm.RunClosure(c)

	return ret, c.UsedGas, err
}

func (c *Closure) Return(ret []byte) []byte {
	// Return the remaining gas to the caller
	c.caller.ReturnGas(c.Gas, c.Price)

	return ret
}

func (c *Closure) UseGas(gas *big.Int) bool {
	if c.Gas.Cmp(gas) < 0 {
		return false
	}

	// Sub the amount of gas from the remaining
	c.Gas.Sub(c.Gas, gas)
	c.UsedGas.Add(c.UsedGas, gas)

	return true
}

// Implement the caller interface
func (c *Closure) ReturnGas(gas, price *big.Int) {
	// Return the gas to the closure
	c.Gas.Add(c.Gas, gas)
	c.UsedGas.Sub(c.UsedGas, gas)
}

func (c *Closure) Object() *monkstate.StateObject {
	return c.object
}

func (c *Closure) Caller() ClosureRef {
	return c.caller
}
