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
	"fmt"
	"io"
)

// DefaultMemSize defines default memory size allocated
// with ToBF/ToMF converter environment.
const DefaultMemSize uint32 = 1024

// Magic is a magic bytes for MF binary file.
const Magic = "\xff\x6d\x66\xfd"

const bf = "+-><[].,"

// ToBF will accept MF code with Write function,
// and write to wrapping Writer interface.
type ToBF struct {
	wr      io.Writer
	rdSize  uint
	misc    [4]byte // magic, memsize storage
	memSize uint32  // at least 32-bit
	sbit    bool    // special bit flag
	scode   byte    // special code
	rdGoal  byte    // bytes limit to read compressed length
}

// NewReader returns new mf.ToBF struct.
func NewReader(wr io.Writer) *ToBF {
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
				r.allocMem(r.miscData())
			}
		case r.rdSize <= r.rdGoal:
			r.misc[(r.rdSize+3)-r.rdGoal] = b
			if r.rdSize == r.rdGoal {
				for i := r.miscData(); i > 0; i-- {
					r.wr.Write([]byte{bf[r.scode]})
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
					r.wr.Write([]byte{bf[r.scode]})
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

func (r *ToBf) miscData() uint32 {
	return uint32(r.misc[0])<<24 | uint32(r.misc[1])<<12 | uint32(r.misc[2])<<8 | uint32(r.misc[3])
}

func (r *ToBF) allocMem(size uint32) {
	//TODO
}
