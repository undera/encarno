import logging
import time

from taurus import KPIReader

logging.basicConfig(level=logging.INFO)

file = "/media/BIG/bzt-artifacts/2022-06-04_09-48-44.020885/encarno.ldjson"

obj = KPIReader(file, logging.getLogger(''), "/dev/null")
for item in obj.datapoints(True):
    logging.info("%s %s", int(time.time()), item)
