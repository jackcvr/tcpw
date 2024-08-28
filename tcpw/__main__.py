import os
import os.path as osp
import sys
import platform

dir = osp.dirname(__file__)
arch = platform.machine().lower()
path = osp.join(dir, f"{osp.basename(dir)}-{arch}")


def main():
	if not osp.exists(path):
		raise RuntimeError(f"Unknown CPU architecture: {arch}")
	os.execv(path, sys.argv)


if __name__=='__main__':
	main()