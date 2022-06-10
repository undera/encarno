import csv
import json
import sys

writer = None
for line in sys.stdin:
    data = json.loads(line)

    if not writer:
        writer = csv.DictWriter(sys.stdout, data.keys())
        writer.writeheader()

    writer.writerow(data)
