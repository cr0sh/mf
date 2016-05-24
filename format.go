// Package mf provides MinFuck code tools:
// converter from/to BrainFuck.
//
// MinFuck 언어는 BrainFuck 언어의 개량 버전입니다.
//
// 연속되는 코드가 많은 부분을 압축할 수 있고,
// BetterBF 구현에 용이하도록 초기 메모리 할당 등을 지정하는
// 메타데이터 부분을 할당했습니다.
//
// MF 바이너리의 첫 4바이트는 Magic(\xff\x6d\x66\xfd),
// 다음 32비트는 할당할 VM 메모리 크기입니다.
// BF에서 MF로 강제 변환한 코드의 경우 Magic은 \xff\x6d\x68\xfd입니다.
//
// 각 BF 코드 1바이트는 MF 코드 1니블로 치환됩니다.
//  +: 0
//  -: 1
//  >: 2
//  <: 3
//  [: 4
//  ]: 5
//  .: 6
//  ,: 7
//
// 주의: [, ] 즉 4, 5 코드는 special code로 대체됩니다. non-special code에서의 4, 5 코드 구현은 표준에서도 정의하지 않습니다.
// 레거시 호환성을 위해 구현할 수 있으나, special code의 4, 5 사용을 권장합니다.
//
// 니블코드의 첫 비트가 1(전체 니블이 8 이상)인 경우, 뒤 3비트는 special code로 취급되어 특수 목적으로 사용됩니다.
//
// 주의: 홀수 번째 니블에 special code가 있다면, 뒤 니블은 버려집니다.
//
// special code가 0~3 중 하나일 경우 +, -, >, < 반복 코드의 압축을 의미합니다.
// 다음 32비트에 코드 반복 횟수를 명시해 압축 코드를 표시합니다.
//
// special code가 4 또는 5인 경우 BF의 [, ] 코드와 유사하게 동작합니다.
// [, ]과 같게 현재 메모리 포인터 값의 zeroness를 검사 후,
// special code 뒤 32비트에서 명시한 위치로 점프합니다.
//
// 주의: ToBF 구조체는 모든 4/5 special code를 단순히 [. ]로 치환합니다.
// 점프 위치로 명시된 곳에 어떤 코드가 있는지는 검사하지 않으니 주의하세요.
// 이러한 불확실성을 악용하여 임의 주소 점프로 사용하면 안 됩니다.
// (BF 코드로 변환했을 때 잘못된 동작을 일으킵니다)
//
// special code가 6인 경우 no-op입니다. 압축 align에 사용됩니다.
//
// 주의: no-op 코드를 일반적 상황에서 직접 삽입할 이유는 없습니다. 예상하지 못한 효과를 일으킬 수 있습니다.
//
// 7은 예약된 special code입니다. (내부 조작, syscall 관련으로 사용될 예정)
//
package mf

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

// DefaultMemSize defines default memory size allocated
// with ToBF/ToMF converter environment.
const DefaultMemSize uint32 = 1024

// Magic is a magic bytes for MF binary file.
const Magic = "\xff\x6d\x66\xfd"

// BFMagic is a magic bytes for BF-converted MF binary file.
const BFMagic = "\xff\x6d\x68\xfd"

const bf = "+-><[].,"

// ToBF will accept MF code with Write function,
// and write to wrapping Writer interface.
type ToBF struct {
	wr      io.Writer
	rdSize  uint32
	misc    [4]byte // magic, memsize storage
	memSize uint32  // at least 32-bit
	sbit    bool    // special bit flag
	scode   byte    // special code
	rdGoal  uint32  // bytes limit to read compressed length
}

// NewBFWriter returns new mf.ToBF struct.
func NewBFWriter(wr io.Writer) *ToBF {
	return &ToBF{wr: wr}
}

// Write implements io.Writer interface.
// Write will write converted BF code from p to wr.
func (r *ToBF) Write(p []byte) (n int, err error) {
	for i, b := range p {
		switch {
		case r.rdSize <= 4:
			if r.rdSize != 4 {
				r.misc[r.rdSize] = b
			} else if string(r.misc[:4]) != Magic {
				return i, fmt.Errorf("Invalid magic 0x%x", r.misc[:4])
			}
		case r.rdSize <= 8:
			if r.rdSize != 8 {
				r.misc[r.rdSize-4] = b
			} else {
				r.wr.Write([]byte("MinFuck compiled code\n"))
				r.allocMem(r.miscData())
			}
		case r.rdSize <= r.rdGoal:
			r.misc[(r.rdSize+3)-r.rdGoal] = b
			if r.rdSize == r.rdGoal {
				for i := r.miscData(); i > 0; i-- {
					r.wr.Write([]byte(bf[r.scode : r.scode+1]))
				}
			}
		default:
			if err := r.processByte(b); err != nil {
				return i, err
			}
			if r.sbit { // special bit
				switch {
				case r.scode < 4: // compressed code
					r.rdGoal = r.rdSize + 4
				case r.scode == 4 || r.scode == 5:
					r.wr.Write([]byte(bf[r.scode : r.scode+1]))
				case r.scode == 6:
					r.sbit = false // no-op
				}
			}
		}
		r.rdSize++
	}
	return
}

func (r *ToBF) processByte(b byte) error {
	if s := b >> 7; s == 0 {
		r.processNibble(b >> 4)
	} else {
		r.sbit = true
		r.scode = (b >> 4) & 7
		return nil
	}
	if s := (b >> 3) & 1; s == 0 {
		r.processNibble(b & 0xf)
	} else {
		r.sbit = true
		r.scode = b & 0x7
	}
	return nil
}

func (r *ToBF) processNibble(n byte) {
	r.wr.Write([]byte{bf[n]})
}

func (r *ToBF) miscData() uint32 {
	return uint32(r.misc[0])<<24 | uint32(r.misc[1])<<12 | uint32(r.misc[2])<<8 | uint32(r.misc[3])
}

func (r *ToBF) allocMem(size uint32) {
	r.wr.Write([]byte(">>+>>+>>+>>+>"))
	r.wr.Write([]byte(strings.Repeat("+", int(size))))
	r.wr.Write([]byte("[[->>+<<]>+>-]<[<<]"))
}

// FromBF converts BF code to MF, and writes to the wrapping Writer.
type FromBF struct {
	wr   *bytes.Buffer
	wrap io.Writer
	buf  byte
	last byte
	dup  uint32
	half bool
}

// NewBFWriter returns new FromBF struct.
func NewBFReader(wr io.Writer, memsize uint32) *FromBF {
	r := new(FromBF)
	r.wr = new(bytes.Buffer)
	r.wrap = wr
	r.wr.Write([]byte(BFMagic))
	r.wr.Write(uint32bytes(memsize))
	return r
}

// Write implements io.Writer interface.
func (r *FromBF) Write(p []byte) (n int, err error) {
	for _, b := range p {
		switch b {
		case 43, 45, 62, 60:
			var t byte
			switch b {
			case 43:
				t = 0
			case 45:
				t = 1
			case 62:
				t = 2
			case 60:
				t = 3
			}
			if t != r.last {
				r.clearDup()
				switch b {
				case 43:
					r.last = 0
				case 45:
					r.last = 1
				case 62:
					r.last = 2
				case 60:
					r.last = 3
				}
				r.dup = 1
			} else {
				r.dup++
			}
		case 91, 93:
			if r.dup > 0 {
				r.clearDup()
			}
			if b == 91 {
				r.writeNibble(8 | 4)
			} else {
				r.writeNibble(8 | 5)
			}
			if r.half {
				r.writeNibble(8 | 6)
			}
			r.wr.Write(make([]byte, 4))
		case 46, 44:
			if r.dup > 0 {
				r.clearDup()
			}
			if b == 46 {
				r.writeNibble(6)
			} else {
				r.writeNibble(7)
			}
		}
	}
	return len(p), nil
}

func (r *FromBF) clearDup() {
	if r.dup > 9 {
		r.writeNibble(8 | r.last)
		if r.half {
			r.writeNibble(14)
		}
		r.wr.Write(uint32bytes(r.dup))
	} else {
		for i := uint32(0); i < r.dup; i++ {
			r.writeNibble(r.last)
		}
	}
	r.dup = 0
}

func (r *FromBF) writeNibble(p byte) error {
	if r.half {
		r.half = false
		_, err := r.wr.Write([]byte{r.buf | (p & 0xf)})
		if err != nil {
			panic(err.Error())
		}
		return err
	} else {
		r.half, r.buf = true, (p&0xf)<<4
		return nil
	}
}

// Close implements io.Closer interface.
func (r *FromBF) Close() error {
	if r.dup > 0 {
		r.clearDup()
	}
	r.cacheJumpOff()
	io.Copy(r.wrap, r.wr)
	return nil
}

func (r *FromBF) cacheJumpOff() {
	s := new(stack)
	s.mem = make([]uint32, 1024)
	buf := r.wr.Bytes()
	fmt.Print(hex.Dump(buf))
	for i := 8; i < len(buf); i++ {
		b := buf[i]
		n1, n2 := b>>4, b&0xf
		if n1 == 0xc || n2 == 0xc {
			s.put(uint32(i))
			i += 4
		} else if n1 == 0xd || n2 == 0xd {
			jmp := s.get()
			fmt.Printf("Loop index pair %2x %2x\n", jmp, i)
			copy(buf[i+1:i+5], uint32bytes(jmp+5))
			copy(buf[jmp+1:jmp+5], uint32bytes(uint32(i)+5))
			i += 4
		}
	}
	fmt.Print(hex.Dump(buf))
	r.wrap.Write(buf)
}

type stack struct {
	mem []uint32
	off int
}

func (s *stack) put(n uint32) {
	if len(s.mem) <= s.off {
		s.mem = append(s.mem, make([]uint32, len(s.mem))...)
	}
	s.mem[s.off] = n
	s.off++
}

func (s *stack) get() uint32 {
	s.off--
	if s.off < 0 {
		panic("invalid stack pointer: tried to get value from empty stack")
	}
	return s.mem[s.off]
}

func uint32bytes(n uint32) []byte {
	return []byte{
		byte(n >> 24),
		byte(n >> 16),
		byte(n >> 8),
		byte(n),
	}
}
