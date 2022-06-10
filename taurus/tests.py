import csv
import logging
import time

from bzt.modules.aggregator import DataPoint

from taurus.encarno import KPIReader

logging.basicConfig(level=logging.INFO)

file = "/tmp/downloads/encarno_results.ldjson"

obj = KPIReader(file, logging.getLogger(''), "/dev/null")
for item in obj.datapoints(True):
    item.recalculate()
    logging.info("%s %s", item[DataPoint.TIMESTAMP], item[DataPoint.CURRENT])
