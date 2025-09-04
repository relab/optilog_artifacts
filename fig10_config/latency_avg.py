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
    latency_event_map = {}
    id_map = {}
    for file_name in find_files(directory_name):
        fp = open(file_name, )
        data = json.load(fp)
        for ele in data:
            if ele['@type'] == "type.googleapis.com/types.LatencyMeasurement":
                timestamp = to_timestamp(ele['Event']['Timestamp'])
                latency = float(ele['Latency'])
                if latency == 0 or latency == 'nan':
                    continue
                id = ele['Event']['ID']
                if id in id_map:
                    id_map[id].append(latency/1000)
                else:
                     id_map[id] = [latency]
                if timestamp in latency_event_map:
                    latency_event_map[timestamp].append(latency)
                else:
                    latency_event_map[timestamp] = [latency]
    result = {}
    max_result = []
    max_length = 0
    for k,v in id_map.items():
        if len(v) > max_length:
            max_length = len(v)
            max_result = v
    print(max_length, max_result)

    for timestamp,values in latency_event_map.items():
        tmp_values =  []
        for value in values:
            tmp_values.append(value)
        if len(tmp_values) != 0:
            result[timestamp] = sum(tmp_values)/len(tmp_values)
    return result

def write_latency_avg(directory_name, latency_data, skip_count):
    header = ["time","latency"]
    with open(directory_name+'/latency_avg.csv', 'w') as f:
        writer = csv.writer(f)
        writer.writerow(header)
        count = 0
        for key in sorted(latency_data.keys()):
            count += 1
            if count <= int(skip_count):
                continue
            writer.writerow([count, latency_data[key]])

if __name__ == '__main__':
    if len(sys.argv) != 3:
        print("usage: python latency_avg.py inputDirectory skipCount")
        exit()
    latency_data = readFromFiles(sys.argv[1])
    write_latency_avg(sys.argv[1], latency_data, sys.argv[2])
