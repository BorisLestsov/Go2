import subprocess
import matplotlib.pyplot as plt
import numpy as np


def bash_command(cmd):
    result = subprocess.Popen(cmd.split(), stdout=subprocess.PIPE).communicate()[0]
    return result[:-1]


def main():
	prob = [0.1, 0.3, 0.5, 0.7]
	vals = []
	iters = 10

	for p in prob:
		s = 0.0
		n = 0

		i = iters
		bash_command('iptables -A INPUT -p udp -i lo -m statistic --mode random --probability '+str(p)+' -j DROP')
		while i > 0:
			res = bash_command('./main 10 30000 3 5 5').split()
			if len(res) >= 3: 
				s += float(res[2])
				n += 1
				i -= 1
		vals.append(s/n if n != 0 else 0.0)
		print vals

		bash_command('iptables -D INPUT -p udp -i lo -m statistic --mode random --probability '+str(p)+' -j DROP')
	print vals


if __name__ == "__main__":
	main()