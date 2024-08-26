import os,os.path as osp,sys,platform
d=osp.dirname(__file__)
a=platform.machine().lower()
p=osp.join(d,f"{osp.basename(d)}-{a}")
def main():
	if not osp.exists(p):raise RuntimeError(f"Unknown CPU architecture: {a}")
	os.execv(p,sys.argv)
if __name__=='__main__':main()