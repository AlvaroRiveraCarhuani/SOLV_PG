import sys

lines = sys.stdin.read().split()
if len(lines) >= 2:
    a = int(lines[0])
    b = int(lines[1])
    print(a * b)
