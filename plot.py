import subprocess
import matplotlib.pyplot as plt
import numpy as np


def bash_command(cmd):
    result = subprocess.Popen(cmd.split(), stdout=subprocess.PIPE).communicate()[0]
    return result[:-1]


def main():
	prob = [0.0, 0.1, 0.2, 0.3, 0.4, 0.5]
	vals = []
	iters = 25

	N = 50 
	port = 30000
	min_d = 5
	max_d = 7
	ttl = 50

	for p in prob:
		s = 0.0
		n = 0
		fails = 0

		i = iters
		bash_command('iptables -A INPUT -p udp -i lo -m statistic --mode random --probability '+str(p)+' -j DROP')
		while i > 0:
			res = bash_command('./main {} {} {} {} {}'.format(N, port, min_d, max_d, ttl)).split()
			if len(res) >= 3: 
				s += float(res[2])
				n += 1
				i -= 1
			else:
				fails += 1
		vals.append(s/n if n != 0 else 0.0)
		print vals

		bash_command('iptables -D INPUT -p udp -i lo -m statistic --mode random --probability '+str(p)+' -j DROP')
	print vals
	xticks = np.arange(len(vals))

	plt.figure(figsize=(10, 8))
	plt.bar(xticks, vals, color='c')
	plt.title("{} nodes, {}..{} degree, {} ttl".format(N, min_d, max_d, ttl))
	plt.xlabel("Packet loss probability")
	plt.ylabel("Time (ticks)")

	plt.savefig("plot.png")

if __name__ == "__main__":
	main()