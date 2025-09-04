import glob
import sys
import json


def find_files(directory_name):
    if directory_name == None or directory_name == "":
        return -1
    files = glob.glob(directory_name+"/*/*.json")
    return files

def readFromFiles(directory_name):
    totalbytes = 0
    totalblocks = 0
    for file_name in find_files(directory_name):
        fp = open(file_name, )
        data = json.load(fp)
        for ele in data:
            if ele['@type'] == "type.googleapis.com/types.SentBytes":
                sentbytes = int(ele['sendEventrd'])
                blocks = int(ele['blocks'])
                if sentbytes == 0 or blocks == 0:
                    continue
                print(sentbytes, blocks)
                totalbytes += sentbytes
                totalblocks += blocks

    print((totalbytes/totalblocks))



if __name__ == '__main__':
    if len(sys.argv) != 2:
        print("usage: python sentdata.py inputDirectory")
        exit()
    readFromFiles(sys.argv[1])

