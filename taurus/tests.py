import logging
import time

from bzt.modules.aggregator import DataPoint, KPISet

from encarno import KPIReaderLDJSON
from taurus.encarno import KPIReaderBinary

logging.basicConfig(level=logging.INFO)

file = "/media/BIG/bzt-artifacts/2022-06-15_13-38-26.285227/encarno_results.ldjson"
sfile = "/media/BIG/bzt-artifacts/2022-06-15_13-38-26.285227/encarno_results.ostr"

start = time.time()
obj = KPIReaderLDJSON(file, logging.getLogger(''), "/dev/null")
# obj = KPIReaderBinary(file, sfile, logging.getLogger(''), "/dev/null")
item = None
for item in obj.datapoints(True):
    logging.info("%s %s", item[DataPoint.TIMESTAMP], item[DataPoint.CURRENT])

elapsed = time.time() - start
cnt = item[DataPoint.CUMULATIVE][''][KPISet.SAMPLE_COUNT]
logging.info("Finished: %s, %s cnt, speed %s", elapsed, cnt, cnt / elapsed)
