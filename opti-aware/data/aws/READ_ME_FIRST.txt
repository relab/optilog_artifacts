(1) Latency data was once obtained as statistical, yearly average from Cloudping https://www.cloudping.co/grid
(2) The latencies in aws-latencies.csv are in a matrix format. The ordering/indexing is given by the order/index in aws-servers.csv
(3) Each latency at index (i,j) represents the RTT (round trip time) from region i to region j (as indexed by aws-servers.csv)
(4) When using inside a emulation/simulation you may convert latencies to one-way latencies