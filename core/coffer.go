package core

import (
	"sync"
	"time"

	"github.com/awnumar/memguard/crypto"
)

var (
	// Interval of time between each verify & rekey cycle.
	Interval uint = 8 // milliseconds
)

// Static allocation for fast random bytes reading.
var buf32 [32]byte

/*
Coffer is a specialized container for securing highly-sensitive, 32 byte values.
*/
type Coffer struct {
	sync.RWMutex

	left  *Buffer // Left partition.
	right *Buffer // Right partition.
}

// NewCoffer is a raw constructor for the *Coffer object.
func NewCoffer() *Coffer {
	// Create a new Coffer object.
	s := new(Coffer)

	// Allocate the partitions.
	s.left, _ = NewBuffer(32)
	s.right, _ = NewBuffer(32)

	// Initialise with a random 32 byte value.
	s.Initialise()

	go func(s *Coffer) {
		for {
			// Sleep for the specified interval.
			time.Sleep(time.Duration(Interval) * time.Millisecond)

			// Check if it's destroyed.
			if s.Destroyed() {
				break
			}

			// Re-key the contents.
			s.Rekey()
		}
	}(s)

	return s
}

/*
Initialise is used to reset the value stored inside a Coffer to a new random 32 byte value, overwriting the old.
*/
func (s *Coffer) Initialise() error {
	// Attain the mutex.
	s.Lock()
	defer s.Unlock()

	// Check if it has been destroyed.
	if !GetBufferState(s.left).IsAlive {
		return ErrDestroyed
	}

	// Overwrite the old value with fresh random bytes.
	if err := crypto.MemScr(s.left.Data); err != nil {
		Panic(err)
	}
	if err := crypto.MemScr(s.right.Data); err != nil {
		Panic(err)
	}

	// left = left XOR hash(right)
	hr := crypto.Hash(s.right.Data)
	for i := range hr {
		s.left.Data[i] ^= hr[i]
	}

	return nil
}

/*
View returns a snapshot of the contents of a Coffer inside a Buffer. As usual the Buffer should be destroyed as soon as possible after use by calling the Destroy method.
*/
func (s *Coffer) View() (*Buffer, error) {
	// Attain a read-only lock.
	s.RLock()
	defer s.RUnlock()

	// Check if it's destroyed.
	if !GetBufferState(s.left).IsAlive {
		return nil, ErrDestroyed
	}

	// Create a new Buffer for the data.
	b, _ := NewBuffer(32) // Will never error as len > 0

	// data = hash(right) XOR left
	h := crypto.Hash(s.right.Data)
	for i := range b.Data {
		b.Data[i] = h[i] ^ s.left.Data[i]
	}

	// Return the view.
	return b, nil
}

/*
Rekey is used to re-key a Coffer. Ideally this should be done at short, regular intervals.
*/
func (s *Coffer) Rekey() {
	// Attain the mutex.
	s.Lock()
	defer s.Unlock()

	// Check if it has been destroyed.
	if !GetBufferState(s.left).IsAlive {
		return
	}

	// Get a new random 32 byte R value.
	if err := crypto.MemScr(buf32[:]); err != nil {
		Panic(err)
	}

	// new_right = old_right XOR randbuf32
	rr := make([]byte, 32)
	for i := range s.right.Data {
		rr[i] = s.right.Data[i] ^ buf32[i]
	}

	// new_left = old_left XOR hash(old_right) XOR hash(new_right)
	hy := crypto.Hash(s.right.Data)
	hrr := crypto.Hash(rr)
	for i := range buf32 {
		s.left.Data[i] ^= hy[i] ^ hrr[i]
	}

	// Copy the new right to the right memory location.
	for i := range s.right.Data {
		s.right.Data[i] = rr[i]
	}
}

/*
Destroy wipes and cleans up all memory related to a Coffer object. Once this method has been called, the Coffer can no longer be used and a new one should be created instead.
*/
func (s *Coffer) Destroy() {
	// Attain the mutex.
	s.Lock()
	defer s.Unlock()

	// Destroy the partitions.
	s.left.Destroy()
	s.right.Destroy()
}

// Destroyed returns a boolean value indicating if a Coffer has been destroyed.
func (s *Coffer) Destroyed() bool {
	s.RLock()
	defer s.RUnlock()

	return !GetBufferState(s.left).IsAlive
}
