"""
MF.py - MinFuck interpreter written in RPython
This code is based on BF.py from example5.py in https://bitbucket.org/brownan/pypy-tutorial.
Thanks to Andrew Brown, an original author of example5.py and awesome PyPy/RPython tutorial.
"""

import os
import sys
import binascii

try:
    from rpython.rlib.jit import JitDriver
except ImportError:
    class JitDriver(object):
        def __init__(self,**kw): pass
        def jit_merge_point(self,**kw): pass
        def can_enter_jit(self,**kw): pass

Magic = "\xff\x6d\x66\xfd"
BFMagic = "\xff\x6d\x68\xfd"

jitdriver = JitDriver(greens=['pc', 'program'], reds=['tape'])

def mainloop(program):
    pc = 8
    tape = None
    
    if len(program) < 8:
        os.write(1, "Invalid MF binary(file too small)\n")
        return 1

    memsize = ord(program[4]) << 24 | ord(program[5]) << 16 | ord(program[6]) << 8 | ord(program[7])
    magic = program[:4]
    if magic == Magic:
        tape = Tape(2*memsize+8)
        for i in range(2, 2*memsize+8,2):
            tape.thetape[i] = 1
    elif magic != BFMagic:
        os.write(1, "Invalid MF binary(magic mismatch)\n")
        return 1
    else:
        tape = Tape(memsize)

    assert type(tape) is Tape
    
    while pc < len(program):
        '''if dbg:
            print hex(pc)[2:], hex(len(program))[2:], tape.thetape, hex(ord(program[pc]))[2:]
            '''
        jitdriver.jit_merge_point(pc=pc, tape=tape, program=program)

        c = ord(program[pc])
        c1, c2 = c>>4, c&0xf        

        spb = -1
        if c1 & 8 == 8:
            spb = c1&7
        elif c2 & 8 == 8 and c2 & 7 != 6:
            tape.control(c1)
            spb = c2&7
        
        if spb == -1:
            tape.control(c1)
            tape.control(c2)
            pc += 1

        else:
            pc += 1
            m = program[pc:pc+4]
            pc += 4
            # print "special: ", spb, data
            if spb == 0:
                tape.inc(ord(m[3]))
            elif spb == 1:
                tape.dec(ord(m[3]))
            else:
                data = abs(ord(m[0])<<24|ord(m[1])<<16|ord(m[2])<<8|ord(m[3]))
                if spb == 2:
                    tape.advance(data)
                elif spb == 3:
                    tape.devance(data)
                elif spb == 4 and tape.get() & 0xff == 0:
                    pc = data
                elif spb == 5 and tape.get() & 0xff != 0:
                    pc = data
    return 0

class Tape(object):
    def __init__(self, size):
        self.thetape = [0] * size
        self.position = 0

    def get(self):
        return self.thetape[self.position]
    def set(self, val):
        self.thetape[self.position] = val
    def inc(self, val):
        self.thetape[self.position] += val
    def dec(self, val):
        self.thetape[self.position] -= val
    def advance(self, val):
        self.position += val
    def devance(self, val):
        self.position -= val

    def control(self, code):
        if code == 0:
                self.inc(1)
        elif code == 1:
            self.dec(1)
        elif code == 2:
            self.advance(1)
        elif code == 3:
            self.devance(1)  
        elif code == 6:
            os.write(1, chr(self.get()))
        elif code == 7:
            # read from stdin
            self.set(ord(os.read(0,1)[0]))


def entry_point(argv):
    try:
        filename = argv[1]
    except IndexError:
        os.write(1, "You must supply a filename\n")
        return 1

    program_contents = ""
    fp = os.open(filename, os.O_RDONLY, 0777)
    while True:
        read = os.read(fp, 4096)
        if len(read) == 0:
            break
        program_contents += read

    os.close(fp)
    return mainloop(program_contents)

def target(*args):
    return entry_point, None
    
def jitpolicy(driver):
    from rpython.jit.codewriter.policy import JitPolicy
    return JitPolicy()

if __name__ == "__main__":
    entry_point(sys.argv)
