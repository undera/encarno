import logging
import time

from taurus.incarne import IncarneKPIReader

logging.basicConfig(level=logging.INFO)

file = "/media/BIG/bzt-artifacts/2022-06-04_09-48-44.020885/incarne_results.ldjson"

obj = IncarneKPIReader(file, logging.getLogger(''), "/dev/null")
for item in obj.datapoints(True):
    logging.info("%s %s", int(time.time()), item)
