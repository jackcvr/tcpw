import os
import sys


def main():
	d = os.path.dirname(__file__)
	os.execv(os.path.join(d, os.path.basename(d)), sys.argv)


if __name__=='__main__':
	main()