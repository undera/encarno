import csv
import logging
import time

from bzt.modules.aggregator import DataPoint, KPISet

from taurus.encarno import KPIReaderBinary

logging.basicConfig(level=logging.INFO)

file = "/media/BIG/bzt-artifacts/2022-06-15_10-37-56.773520/encarno_results.bin"

start = time.time()
obj = KPIReaderBinary(file, logging.getLogger(''), "/dev/null")
item = None
for item in obj.datapoints(True):
    item.recalculate()
    logging.info("%s %s", item[DataPoint.TIMESTAMP], item[DataPoint.CURRENT])

elapsed = time.time() - start
cnt = item[DataPoint.CUMULATIVE][''][KPISet.SAMPLE_COUNT]
logging.info("Finished: %s, %s cnt, speed %s", elapsed, cnt, cnt / elapsed)
