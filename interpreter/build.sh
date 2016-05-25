rm -Rf bin
mkdir bin

TOOLCHAIN='time /home/cr0sh/pypy-src/rpython/bin/rpython --no-log '
echo "TOOLCHAIN: $TOOLCHAIN"

echo "building bf.py without jit"
$TOOLCHAIN bf.py
mv bf-c bin/bf-no-jit

echo "building bf.py with jit"
$TOOLCHAIN --opt=jit bf.py
mv bf-c bin/bf-jit

echo "building mf.py without jit"
$TOOLCHAIN mf.py
mv mf-c bin/mf-no-jit

echo "building mf.py with jit"
$TOOLCHAIN --opt=jit mf.py
mv mf-c bin/mf-jit

echo "benchmark bf(pypy)"
time pypy bf.py ../bf/bench.bf

echo "benchmark bf(compiled, no jit)"
time bin/bf-no-jit ../bf/bench.bf

echo "benchmark bf(compiled, with jit)"
time bin/bf-jit ../bf/bench.bf

echo "benchmark mf(pypy)"
time pypy mf.py ../mf/bench.mf

echo "benchmark mf(compiled, no jit)"
time bin/mf-no-jit ../mf/bench.mf

echo "benchmark mf(compiled, with jit)"
time bin/mf-jit ../mf/bench.mf

echo "benchmark bf(cpython)"
time python2 bf.py ../bf/bench.bf

echo "benchmark mf(cpython)"
time python2 mf.py ../mf/bench.mf
