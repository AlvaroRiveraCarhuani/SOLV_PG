import sys

def main():
    lines = sys.stdin.read().split()
    if len(lines) >= 2:
        a = int(lines[0])
        b = int(lines[1])
        print(a + b)

if __name__ == '__main__':
    main()
