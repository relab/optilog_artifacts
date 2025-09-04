import glob
import sys
import json
import csv
from dateutil.parser import isoparse


def find_files(directory_name):
    if directory_name == None or directory_name == "":
        return -1
    files = glob.glob(directory_name+"/*/*.json")
    return files

def to_timestamp(timestamp_str):
    return int(isoparse(timestamp_str).timestamp())

def readFromFiles(directory_name):
    throughput_event_map = {}
    id_map = {}
    for file_name in find_files(directory_name):
        fp = open(file_name, )
        data = json.load(fp)
        for ele in data:
            if ele['@type'] == "type.googleapis.com/types.ThroughputMeasurement":
                timestamp = to_timestamp(ele['Event']['Timestamp'])
                commands = ele['Commands']
                duration_str = ele['Duration'][:-1] #remove s
                interval = float(duration_str)
                throughput = float(commands)/interval
                id = ele['Event']['ID']
                if id in id_map:
                    id_map[id].append(throughput)
                else:
                     id_map[id] = [throughput]
                if timestamp in throughput_event_map:
                    throughput_event_map[timestamp].append(throughput)
                else:
                    throughput_event_map[timestamp] = [throughput]
    result = {}
    max_result = []
    max_length = 0
    for k,v in id_map.items():
        if len(v) > max_length:
            max_length = len(v)
            max_result = v
    print(max_length, max_result)

    for timestamp,values in throughput_event_map.items():
        tmp_values =  []
        for value in values:
            tmp_values.append(value)
        if len(tmp_values) != 0:
            result[timestamp] = sum(tmp_values)/len(tmp_values)
    return result

def write_throughput_avg(directory_name, throughput_data, skip_count):
    header = ["time","commands"]
    with open(directory_name+'/throughput_avg.csv', 'w') as f:
        writer = csv.writer(f)
        writer.writerow(header)
        count = 0
        for key in sorted(throughput_data.keys()):
            count += 1
            if count <= int(skip_count):
                continue
            writer.writerow([count, throughput_data[key]])

if __name__ == '__main__':
    if len(sys.argv) != 3:
        print("usage: python throughput_avg.py inputDirectory skipCount")
        exit()
    throughput_data = readFromFiles(sys.argv[1])
    write_throughput_avg(sys.argv[1], throughput_data, sys.argv[2])
